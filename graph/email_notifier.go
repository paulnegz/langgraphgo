package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"time"
)

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SenderEmail  string
	SenderPass   string
	SenderName   string
}

type EmailNotifier struct {
	config *EmailConfig
}

type NotificationData struct {
	RecipientName string
	LibraryName   string
	Version       string
	Changes       []string
	GitHubURL     string
	Date          string
}

func NewEmailNotifier() (*EmailNotifier, error) {
	config := &EmailConfig{
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     os.Getenv("SMTP_PORT"),
		SenderEmail:  os.Getenv("SENDER_EMAIL"),
		SenderPass:   os.Getenv("SENDER_PASS"),
		SenderName:   os.Getenv("SENDER_NAME"),
	}
	
	if config.SMTPHost == "" {
		config.SMTPHost = "smtp.gmail.com"
	}
	if config.SMTPPort == "" {
		config.SMTPPort = "587"
	}
	if config.SenderName == "" {
		config.SenderName = "LangGraphGo Team"
	}
	
	return &EmailNotifier{config: config}, nil
}

func (e *EmailNotifier) SendNotification(recipientEmail, recipientName string, data NotificationData) error {
	if data.Date == "" {
		data.Date = time.Now().Format("January 2, 2006")
	}
	
	subject := fmt.Sprintf("LangGraphGo %s - New Updates Available", data.Version)
	body, err := e.generateEmailBody(data)
	if err != nil {
		return fmt.Errorf("failed to generate email body: %w", err)
	}
	
	message := e.buildMessage(recipientEmail, subject, body)
	
	auth := smtp.PlainAuth("", e.config.SenderEmail, e.config.SenderPass, e.config.SMTPHost)
	
	addr := fmt.Sprintf("%s:%s", e.config.SMTPHost, e.config.SMTPPort)
	err = smtp.SendMail(addr, auth, e.config.SenderEmail, []string{recipientEmail}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	return nil
}

func (e *EmailNotifier) generateEmailBody(data NotificationData) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background-color: #4CAF50;
            color: white;
            padding: 20px;
            text-align: center;
            border-radius: 5px;
        }
        .content {
            padding: 20px;
            background-color: #f4f4f4;
            margin-top: 20px;
            border-radius: 5px;
        }
        .changes-list {
            background-color: white;
            padding: 15px;
            border-radius: 5px;
            margin: 10px 0;
        }
        .changes-list ul {
            margin: 10px 0;
            padding-left: 20px;
        }
        .cta-button {
            display: inline-block;
            padding: 10px 20px;
            background-color: #4CAF50;
            color: white;
            text-decoration: none;
            border-radius: 5px;
            margin-top: 20px;
        }
        .footer {
            text-align: center;
            margin-top: 30px;
            font-size: 12px;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>LangGraphGo {{.Version}} Release</h1>
    </div>
    
    <div class="content">
        <p>Hello {{if .RecipientName}}{{.RecipientName}}{{else}}Developer{{end}},</p>
        
        <p>We're excited to announce new updates to <strong>{{.LibraryName}}</strong>!</p>
        
        <div class="changes-list">
            <h3>What's New:</h3>
            <ul>
                {{range .Changes}}
                <li>{{.}}</li>
                {{end}}
            </ul>
        </div>
        
        <p>These enhancements make LangGraphGo more powerful and easier to use for building stateful, multi-actor applications with LLMs.</p>
        
        <center>
            <a href="{{.GitHubURL}}" class="cta-button">View on GitHub</a>
        </center>
    </div>
    
    <div class="footer">
        <p>{{.Date}}</p>
        <p>You're receiving this because you've shown interest in LangGraphGo.</p>
        <p>Thank you for being part of our community!</p>
    </div>
</body>
</html>
`
	
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}
	
	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

func (e *EmailNotifier) buildMessage(to, subject, body string) string {
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", e.config.SenderName, e.config.SenderEmail)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"
	
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body
	
	return message
}

func (e *EmailNotifier) SendBatchNotifications(recipients []map[string]string, data NotificationData) ([]string, []string) {
	var successful []string
	var failed []string
	
	for _, recipient := range recipients {
		email := recipient["email"]
		name := recipient["name"]
		
		if email == "" {
			continue
		}
		
		data.RecipientName = name
		err := e.SendNotification(email, name, data)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", email, err))
		} else {
			successful = append(successful, email)
		}
		
		time.Sleep(1 * time.Second)
	}
	
	return successful, failed
}

func (e *EmailNotifier) LoadRecipientsFromJSON(filename string) ([]map[string]string, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	var recipients []map[string]string
	err = json.Unmarshal(file, &recipients)
	if err != nil {
		return nil, err
	}
	
	return recipients, nil
}