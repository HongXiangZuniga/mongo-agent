// Package web implementa el adapter de entrada HTTP para la interfaz de
// chat basada en HTML + htmx.
package web

import (
	agent "github.com/HongXiangZuniga/mongo-agent/pkg/agent"
)

type tabViewModel struct {
	SessionID string
	Title     string
	Active    bool
}

type messageViewModel struct {
	Role     string
	Segments []messageSegment
}

type chatPanelViewModel struct {
	SessionID      string
	Messages       []messageViewModel
	Error          string
	EmptyStateHint string
}

type pageViewModel struct {
	Tabs      []tabViewModel
	ActiveTab chatPanelViewModel
}

// toTabViewModels convierte SessionSummary a view models de pestañas.
func toTabViewModels(summaries []agent.SessionSummary, activeSessionID string) []tabViewModel {
	tabs := make([]tabViewModel, 0, len(summaries))
	for _, s := range summaries {
		tabs = append(tabs, tabViewModel{
			SessionID: s.SessionID,
			Title:     s.Title,
			Active:    s.SessionID == activeSessionID,
		})
	}
	return tabs
}

// toMessageViewModels convierte Message a view models de mensajes,
// segmentando el contenido en texto plano y tablas Markdown.
func toMessageViewModels(messages []agent.Message) []messageViewModel {
	viewModels := make([]messageViewModel, 0, len(messages))
	for _, m := range messages {
		viewModels = append(viewModels, messageViewModel{
			Role:     string(m.Role),
			Segments: parseMessageContent(m.Content),
		})
	}
	return viewModels
}
