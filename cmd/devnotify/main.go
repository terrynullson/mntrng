package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

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
		agentName        string
		module           string
		commit           string
		mood             string
		summary          summaryFlags
		thoughts         thoughtFlags
		summaryFile      string
		moodFile         string
		readSummaryFromGit bool
		testSend          bool
	)

	flag.StringVar(&agentName, "agent", "BackendAgent", "agent name")
	flag.StringVar(&module, "module", "", "module name")
	flag.StringVar(&commit, "commit", "", "commit hash")
	flag.StringVar(&mood, "mood", "Бодро", "completion mood")
	flag.Var(&summary, "summary", "summary line (can be repeated)")
	flag.Var(&thoughts, "thought", "thought line (can be repeated, up to 2)")
	flag.StringVar(&summaryFile, "summaryFile", "", "read summary from UTF-8 file")
	flag.StringVar(&moodFile, "moodFile", "", "read mood from UTF-8 file")
	flag.BoolVar(&readSummaryFromGit, "readSummaryFromGit", false, "run git log -1 --format=%s with UTF-8 and use as summary (for post-commit hook)")
	flag.BoolVar(&testSend, "test", false, "send a test message with Cyrillic to verify Telegram encoding")
	flag.Parse()

	if testSend {
		agentName = "Test"
		module = "test"
		commit = "test"
		summary = []string{"Проверка кириллицы в DevLog: сводка и настроение приходят корректно."}
		mood = "Коммит прошел"
	} else {
		if strings.TrimSpace(module) == "" {
			log.Fatal("module is required")
		}
		if strings.TrimSpace(commit) == "" {
			log.Fatal("commit is required")
		}
	}

	if !testSend && readSummaryFromGit {
		s, err := gitLogSubjectUTF8()
		if err != nil {
			log.Printf("git log subject: %v", err)
		} else {
			s = strings.TrimSpace(s)
			if s != "" {
				summary = []string{s}
				// derive module from subject: "api: add foo" -> api
				if idx := strings.Index(s, ":"); idx > 0 {
					if m := strings.TrimSpace(s[:idx]); m != "" {
						module = m
					}
				}
			}
		}
		mood = "Коммит прошел"
	} else if summaryFile != "" {
		s, err := readUTF8File(summaryFile)
		if err != nil {
			log.Fatalf("read summary file: %v", err)
		}
		summary = []string{strings.TrimSpace(s)}
	}
	if moodFile != "" {
		s, err := readUTF8File(moodFile)
		if err != nil {
			log.Fatalf("read mood file: %v", err)
		}
		mood = strings.TrimSpace(s)
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

// gitLogSubjectUTF8 runs "git log -1 --format=%s" with UTF-8 output and returns the subject. Avoids PowerShell/console encoding.
func gitLogSubjectUTF8() (string, error) {
	cmd := exec.Command("git", "-c", "i18n.logOutputEncoding=utf-8", "log", "-1", "--format=%s")
	cmd.Env = append(os.Environ(), "LC_ALL=C.UTF-8", "LANG=C.UTF-8")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	b := bytes.TrimSpace(out)
	if utf8.Valid(b) {
		return string(b), nil
	}
	dec := charmap.Windows1251.NewDecoder()
	decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(b), dec))
	if err != nil {
		return string(b), nil
	}
	return string(decoded), nil
}

// readUTF8File reads file and returns content as UTF-8. If bytes are not valid UTF-8, decodes as Windows-1251 (typical on Russian Windows).
func readUTF8File(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if utf8.Valid(b) {
		return string(b), nil
	}
	dec := charmap.Windows1251.NewDecoder()
	out, err := io.ReadAll(transform.NewReader(bytes.NewReader(b), dec))
	if err != nil {
		return string(b), nil
	}
	return string(out), nil
}
