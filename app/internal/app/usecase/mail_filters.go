package usecase

import (
	"path/filepath"
	"strings"

	"electronic-digital-signature/internal/domain/model"
)

type CompositeMessageFilter struct {
	filters []MessageFilter
}

func NewCompositeMessageFilter(filters ...MessageFilter) *CompositeMessageFilter {
	compact := make([]MessageFilter, 0, len(filters))
	for _, filter := range filters {
		if filter != nil {
			compact = append(compact, filter)
		}
	}

	return &CompositeMessageFilter{filters: compact}
}

func (f *CompositeMessageFilter) Match(message model.MailMessage) bool {
	for _, filter := range f.filters {
		if !filter.Match(message) {
			return false
		}
	}

	return true
}

type RecipientFilter struct {
	recipientEmail string
}

func NewRecipientFilter(recipientEmail string) *RecipientFilter {
	return &RecipientFilter{recipientEmail: strings.TrimSpace(strings.ToLower(recipientEmail))}
}

func (f *RecipientFilter) Match(message model.MailMessage) bool {
	if f.recipientEmail == "" {
		return true
	}

	for _, recipient := range message.To {
		if strings.EqualFold(strings.TrimSpace(recipient.Email), f.recipientEmail) {
			return true
		}
	}

	return false
}

type SubjectPrefixFilter struct {
	prefix string
}

func NewSubjectPrefixFilter(prefix string) *SubjectPrefixFilter {
	return &SubjectPrefixFilter{prefix: strings.TrimSpace(prefix)}
}

func (f *SubjectPrefixFilter) Match(message model.MailMessage) bool {
	if f.prefix == "" {
		return true
	}

	return strings.HasPrefix(strings.TrimSpace(message.Subject), f.prefix)
}

type AttachmentSuffixFilter struct {
	suffix string
}

func NewAttachmentSuffixFilter(suffix string) *AttachmentSuffixFilter {
	return &AttachmentSuffixFilter{suffix: strings.TrimSpace(strings.ToLower(suffix))}
}

func (f *AttachmentSuffixFilter) Match(message model.MailMessage) bool {
	if f.suffix == "" {
		return len(message.Attachments) > 0
	}

	for _, attachment := range message.Attachments {
		if strings.HasSuffix(strings.ToLower(filepath.Base(attachment.FileName)), f.suffix) {
			return true
		}
	}

	return false
}
