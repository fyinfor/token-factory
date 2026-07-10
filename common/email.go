package common

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/smtp"
	"slices"
	"strings"
	"time"
)

// EmailAttachment 邮件附件。
type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

func generateMessageID() (string, error) {
	split := strings.Split(SMTPFrom, "@")
	if len(split) < 2 {
		return "", fmt.Errorf("invalid SMTP account")
	}
	domain := strings.Split(SMTPFrom, "@")[1]
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), GetRandomString(12), domain), nil
}

func SendEmail(subject string, receiver string, content string) error {
	return SendEmailWithAttachments(subject, receiver, content, nil)
}

func SendEmailWithAttachments(subject string, receiver string, content string, attachments []EmailAttachment) error {
	if SMTPFrom == "" { // for compatibility
		SMTPFrom = SMTPAccount
	}
	id, err2 := generateMessageID()
	if err2 != nil {
		return err2
	}
	if SMTPServer == "" && SMTPAccount == "" {
		return fmt.Errorf("SMTP 服务器未配置")
	}
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	mail, err := buildEmailMessage(receiver, encodedSubject, id, content, attachments)
	if err != nil {
		return err
	}
	auth := smtp.PlainAuth("", SMTPAccount, SMTPToken, SMTPServer)
	addr := fmt.Sprintf("%s:%d", SMTPServer, SMTPPort)
	to := strings.Split(receiver, ";")
	var sendErr error
	if SMTPPort == 465 || SMTPSSLEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         SMTPServer,
		}
		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", SMTPServer, SMTPPort), tlsConfig)
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, SMTPServer)
		if err != nil {
			return err
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(SMTPFrom); err != nil {
			return err
		}
		receiverEmails := strings.Split(receiver, ";")
		for _, receiver := range receiverEmails {
			if err = client.Rcpt(receiver); err != nil {
				return err
			}
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(mail)
		if err != nil {
			return err
		}
		sendErr = w.Close()
	} else if isOutlookServer(SMTPAccount) || slices.Contains(EmailLoginAuthServerList, SMTPServer) {
		auth = LoginAuth(SMTPAccount, SMTPToken)
		sendErr = smtp.SendMail(addr, auth, SMTPFrom, to, mail)
	} else {
		sendErr = smtp.SendMail(addr, auth, SMTPFrom, to, mail)
	}
	if sendErr != nil {
		SysError(fmt.Sprintf("failed to send email to %s: %v", receiver, sendErr))
	}
	return sendErr
}

func buildEmailMessage(receiver, encodedSubject, messageID, htmlContent string, attachments []EmailAttachment) ([]byte, error) {
	var body bytes.Buffer
	if len(attachments) == 0 {
		body.WriteString(fmt.Sprintf("To: %s\r\n"+
			"From: %s <%s>\r\n"+
			"Subject: %s\r\n"+
			"Date: %s\r\n"+
			"Message-ID: %s\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n",
			receiver, SystemName, SMTPFrom, encodedSubject, time.Now().Format(time.RFC1123Z), messageID, htmlContent))
		return body.Bytes(), nil
	}

	writer := multipart.NewWriter(&body)
	body.WriteString(fmt.Sprintf("To: %s\r\n"+
		"From: %s <%s>\r\n"+
		"Subject: %s\r\n"+
		"Date: %s\r\n"+
		"Message-ID: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: multipart/mixed; boundary=%s\r\n\r\n",
		receiver, SystemName, SMTPFrom, encodedSubject, time.Now().Format(time.RFC1123Z), messageID, writer.Boundary()))

	htmlPart, err := writer.CreatePart(map[string][]string{
		"Content-Type": {"text/html; charset=UTF-8"},
	})
	if err != nil {
		return nil, err
	}
	if _, err = htmlPart.Write([]byte(htmlContent)); err != nil {
		return nil, err
	}

	for _, attachment := range attachments {
		if len(attachment.Data) == 0 {
			continue
		}
		filename := strings.TrimSpace(attachment.Filename)
		if filename == "" {
			filename = "attachment.pdf"
		}
		contentType := strings.TrimSpace(attachment.ContentType)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		filePart, err := writer.CreatePart(map[string][]string{
			"Content-Type":              {contentType},
			"Content-Transfer-Encoding": {"base64"},
			"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, mime.QEncoding.Encode("utf-8", filename))},
		})
		if err != nil {
			return nil, err
		}
		encoder := base64.NewEncoder(base64.StdEncoding, filePart)
		if _, err = encoder.Write(attachment.Data); err != nil {
			return nil, err
		}
		if err = encoder.Close(); err != nil {
			return nil, err
		}
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}
