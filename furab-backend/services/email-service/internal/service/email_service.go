// Package service implements the business logic for email-service.
package service

import (
	"context"
	"errors"
	"time"

	"furab-backend/services/email-service/internal/model"
	"furab-backend/services/email-service/internal/repository"

	"github.com/google/uuid"
)

var eventTemplates = map[string]struct {
	subject string
	body    string
}{
	"payment.success": {subject: "Invoice", body: "Detail transaksi"},
}

// EmailSender defines outbound email provider client behavior.
type EmailSender interface {
	Send(ctx context.Context, receiverEmail, subject, body string) error
}

// EmailService defines the interface for email-service business logic.
type EmailService interface {
	SendEmail(ctx context.Context, req model.SendEmailRequest) (*model.EmailResponse, error)
	SendEmailWithResult(ctx context.Context, req model.SendEmailRequest) (*model.EmailResult, error)
	TriggerEventEmail(ctx context.Context, req model.EventEmailRequest) (*model.EmailResponse, error)
}

// emailServiceImpl is the concrete implementation of EmailService.
type emailServiceImpl struct {
	repo   repository.EmailRepository
	sender EmailSender
}

// NewEmailService creates a new EmailService.
func NewEmailService(repo repository.EmailRepository, sender EmailSender) EmailService {
	return &emailServiceImpl{
		repo:   repo,
		sender: sender,
	}
}

func (s *emailServiceImpl) SendEmail(ctx context.Context, req model.SendEmailRequest) (*model.EmailResponse, error) {
	if req.TemplateID != "" {
		template, err := s.repo.GetTemplateByID(ctx, req.TemplateID)
		if err != nil {
			return nil, err
		}
		if template == nil {
			return nil, errors.New("template not found")
		}
		req.Subject = template.Subject
		req.Body = template.Body
	}

	if req.ReceiverEmail == "" || req.Subject == "" || req.Body == "" {
		return nil, errors.New("invalid request")
	}

	emailID := uuid.New().String()
	now := time.Now().UTC()

	log := model.EmailLog{
		EmailID:       emailID,
		ReceiverEmail: req.ReceiverEmail,
		Subject:       req.Subject,
		Status:        "sent",
		Timestamp:     now,
		ReceiverID:    req.ReceiverID,
		ReferenceID:   req.ReferenceID,
	}

	if err := s.repo.SaveEmailLog(ctx, log); err != nil {
		return nil, err
	}

	if err := s.sender.Send(ctx, req.ReceiverEmail, req.Subject, req.Body); err != nil {
		log.Status = "failed"
		s.repo.SaveEmailLog(ctx, log) // update status to failed
		return nil, err
	}

	return &model.EmailResponse{
		EmailID:       emailID,
		ReceiverEmail: req.ReceiverEmail,
		Subject:       req.Subject,
		Status:        "sent",
		Timestamp:     now,
	}, nil
}

func (s *emailServiceImpl) TriggerEventEmail(ctx context.Context, req model.EventEmailRequest) (*model.EmailResponse, error) {
	template, ok := eventTemplates[req.EventType]
	if !ok {
		return nil, errors.New("invalid event")
	}

	return s.SendEmail(ctx, model.SendEmailRequest{
		ReceiverEmail: req.ReceiverEmail,
		Subject:       template.subject,
		Body:          template.body,
		ReceiverID:    req.ReceiverID,
		ReferenceID:   req.ReferenceID,
		Metadata:      req.Metadata,
	})
}

func (s *emailServiceImpl) SendEmailWithResult(ctx context.Context, req model.SendEmailRequest) (*model.EmailResult, error) {
	res, err := s.SendEmail(ctx, req)
	if err != nil {
		return &model.EmailResult{
			Status:  "failed",
			Message: err.Error(),
			Data:    nil,
		}, err
	}
	return &model.EmailResult{
		Status:  "success",
		Message: "email berhasil dikirim",
		Data:    res,
	}, nil
}
