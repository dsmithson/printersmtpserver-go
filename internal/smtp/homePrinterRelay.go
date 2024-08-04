package smtp

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"printersmtpserver/internal/config"
	"printersmtpserver/internal/utils"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// HomePrinterRelay represents the SMTP relay server
type HomePrinterRelay struct {
	listener  net.Listener
	settings  *config.Config
	isRunning bool
}

// NewHomePrinterRelay creates a new HomePrinterRelay instance
func NewHomePrinterRelay(settings *config.Config) *HomePrinterRelay {
	return &HomePrinterRelay{settings: settings}
}

// Startup starts the SMTP relay server
func (h *HomePrinterRelay) Startup() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", h.settings.SmtpPort))
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	h.listener = listener
	h.isRunning = true

	go h.acceptConnections()
	return nil
}

// Shutdown stops the SMTP relay server
func (h *HomePrinterRelay) Shutdown() {
	h.isRunning = false
	if h.listener != nil {
		h.listener.Close()
	}
}

// acceptConnections handles incoming connections
func (h *HomePrinterRelay) acceptConnections() {
	for h.isRunning {
		conn, err := h.listener.Accept()
		if err != nil {
			if !h.isRunning {
				return
			}
			logrus.WithError(err).Warn("Failed to accept connection.")
			continue
		}

		go h.handleConnection(conn)
	}
}

// handleConnection processes a single client connection
func (h *HomePrinterRelay) handleConnection(conn net.Conn) {
	defer conn.Close()

	clientIP := conn.RemoteAddr().String()
	logrus.Infof("Client connected - %s", clientIP)

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	if _, err := writer.WriteString("220 localhost -- Knightware proxy server\r\n"); err != nil {
		logrus.WithError(err).Warn("Failed to send greeting.")
		return
	}
	writer.Flush()

	var saveToSubFolderName string
	// readBackBuffer := bytes.NewBuffer(nil)

	for h.isRunning {
		msg, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			logrus.WithError(err).Warn("Failed to read from client.")
			return
		}

		msg = strings.TrimSpace(msg)

		if msg == "QUIT" || msg == "quit" {
			logrus.Infof("Closing connection to %s", clientIP)
			break
		}

		if strings.HasPrefix(msg, "EHLO") {
			logrus.Debugf("Received EHLO from: %s", msg[5:])
			_, err := writer.WriteString("250 OK\r\n")
			if err != nil {
				logrus.WithError(err).Warn("Failed to write response.")
				break
			}
		} else if strings.HasPrefix(msg, "RCPT TO:") {
			emailTo := utils.CleanEmailString(msg, "RCPT TO:")
			logrus.Debugf("Received email to: %s", emailTo)

			saveToSubFolderName = utils.ConvertEmailRecipientToFolderName(emailTo)
			_, err := writer.WriteString("250 OK\r\n")
			if err != nil {
				logrus.WithError(err).Warn("Failed to write response.")
				break
			}
		} else if strings.HasPrefix(msg, "MAIL FROM:") {
			emailFrom := utils.CleanEmailString(msg, "MAIL FROM:")
			logrus.Debugf("Received email from: %s", emailFrom)
			_, err := writer.WriteString("250 OK\r\n")
			if err != nil {
				logrus.WithError(err).Warn("Failed to write response.")
				break
			}
		} else if strings.HasPrefix(msg, "DATA") {
			tempFile, err := os.CreateTemp("", "smtp_*")
			if err != nil {
				logrus.WithError(err).Warn("Failed to create temp file.")
				continue
			}
			defer tempFile.Close()

			_, err = writer.WriteString("354 Start mail input; end with <CR><LF>.<CR><LF>\r\n")
			if err != nil {
				logrus.WithError(err).Warn("Failed to write response.")
				break
			}
			writer.Flush()

			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					logrus.WithError(err).Warn("Failed to read DATA from client.")
					break
				}
				if strings.TrimSpace(line) == "." {
					break
				}
				if strings.HasPrefix(line, "..") {
					line = line[1:]
				}
				_, err = tempFile.WriteString(line)
				if err != nil {
					logrus.WithError(err).Warn("Failed to write to temp file")
					break
				}
			}

			_, err = writer.WriteString("250 OK\r\n")
			if err != nil {
				logrus.WithError(err).Warn("Failed to write response.")
				break
			}
			err = writer.Flush()
			if err != nil {
				logrus.WithError(err).Warn("Failed to flush response.")
				break
			}

			savePath := filepath.Join(h.settings.FilePath, saveToSubFolderName)
			go processData(tempFile.Name(), savePath)
		} else {
			logrus.Warnf("Unrecognized data: %s", msg)
		}

		writer.Flush()
	}

	logrus.Infof("Client disconnected - %s", clientIP)
}

// processData processes the email data and saves attachments
func processData(fileName, savePath string) {
	defer os.Remove(fileName)

	const startLineText = "Content-Transfer-Encoding: base64"

	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		err := os.MkdirAll(savePath, 0755)
		if err != nil {
			logrus.WithError(err).Warn("Failed to create folder.")
			return
		}
	}

	startTextFound := false
	isReading := false
	var builder strings.Builder

	file, err := os.Open(fileName)
	if err != nil {
		logrus.WithError(err).Warn("Failed to open file for processing.")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !isReading {
			if line == startLineText {
				startTextFound = true
			} else if startTextFound && line == "" {
				isReading = true
			}
		} else {
			if line == "" {
				now := time.Now()
				newFileName := fmt.Sprintf("%d-%02d-%02d %02d-%02d-%02d - Scan.pdf",
					now.Year(), now.Month(), now.Day(),
					now.Hour(), now.Minute(), now.Second())

				newFile := filepath.Join(savePath, newFileName)
				data, err := base64.StdEncoding.DecodeString(builder.String())
				if err != nil {
					logrus.WithError(err).Warn("Failed to decode base64 data.")
					return
				}
				if err := os.WriteFile(newFile, data, 0644); err != nil {
					logrus.WithError(err).Warn("Failed to write file.")
				}
				break
			}
			builder.WriteString(line)
		}
	}

	if err := scanner.Err(); err != nil {
		logrus.WithError(err).Warn("Error scanning file.")
	}
}
