package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/config"
	"github.com/example/hls-monitoring-platform/internal/telegram"
)

type summaryFlags []string

func (s *summaryFlags) String() string {
	return strings.Join(*s, ";")
}

func (s *summaryFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type thoughtFlags []string

func (t *thoughtFlags) String() string {
	return strings.Join(*t, ";")
}

func (t *thoughtFlags) Set(value string) error {
	*t = append(*t, value)
	return nil
}

func main() {
	var (
		agentName string
		module    string
		commit    string
		mood      string
		summary   summaryFlags
		thoughts  thoughtFlags
	)

	flag.StringVar(&agentName, "agent", "BackendAgent", "agent name")
	flag.StringVar(&module, "module", "", "module name")
	flag.StringVar(&commit, "commit", "", "commit hash")
	flag.StringVar(&mood, "mood", "Бодро", "completion mood")
	flag.Var(&summary, "summary", "summary line (can be repeated)")
	flag.Var(&thoughts, "thought", "thought line (can be repeated, up to 2)")
	flag.Parse()

	if strings.TrimSpace(module) == "" {
		log.Fatal("module is required")
	}
	if strings.TrimSpace(commit) == "" {
		log.Fatal("commit is required")
	}

	cfg := config.LoadDevLogNotifyConfig()
	if !cfg.Enabled {
		log.Printf("dev log notifier is disabled: module=%s commit=%s", module, commit)
		return
	}
	if err := cfg.Validate(); err != nil {
		log.Printf("dev log notifier configuration is invalid: %v", err)
		return
	}

	notifier := telegram.NewDevLogNotifier(
		cfg.Enabled,
		cfg.Token,
		cfg.ChatID,
		&http.Client{Timeout: cfg.HTTPTimeout},
	)

	err := notifier.NotifyCompletion(context.Background(), telegram.DevLogPayload{
		Module:   module,
		Agent:    agentName,
		Commit:   commit,
		Status:   "УСПЕХ",
		Summary:  summary,
		Mood:     mood,
		Thoughts: thoughts,
	})
	if err != nil {
		log.Printf("dev log notifier send failed: %v", err)
		return
	}

	log.Printf("dev log notifier sent: module=%s commit=%s", module, commit)
}
