package codex

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode"

	"enoch/internal/config"
	"enoch/internal/logging"
)

type Client struct {
	command    string
	args       []string
	promptMode string
	timeout    time.Duration
	workdir    string
	useTTY     bool
	disableCPR bool
	progress   time.Duration
	logger     *logging.Logger
}

func New(cfg config.Config, logger *logging.Logger) *Client {
	return &Client{
		command:    cfg.CodexCommand,
		args:       cfg.CodexArgs,
		promptMode: cfg.CodexPromptMode,
		timeout:    cfg.CodexTimeout,
		workdir:    cfg.CodexWorkdir,
		useTTY:     cfg.CodexUseTTY,
		disableCPR: cfg.CodexDisableCPR,
		progress:   cfg.CodexProgressInterval,
		logger:     logger,
	}
}

func (c *Client) Run(prompt string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		if c.logger != nil {
			c.logger.Errorf("codex prompt is empty")
		}
		return "", fmt.Errorf("empty prompt")
	}

	args := make([]string, 0, len(c.args)+1)
	args = append(args, c.args...)

	promptPreview := truncatePrompt(prompt, 160)
	if c.logger != nil {
		c.logger.Debugf("codex invoke: cmd=%s args=%q mode=%s tty=%t prompt=%q", c.command, c.args, c.promptMode, c.useTTY, promptPreview)
	}

	if c.promptMode == "arg" {
		var used bool
		args, used = replacePromptPlaceholder(args, prompt)
		if !used {
			args = append(args, prompt)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	if c.useTTY {
		output, err := c.runWithScript(ctx, prompt, args, promptPreview)
		if err != nil && isTTYError(err) {
			if c.logger != nil {
				c.logger.Warnf("codex tty error, retrying without tty: %v", err)
			}
			return c.runWithoutTTY(ctx, prompt, args, promptPreview)
		}
		return output, err
	}
	return c.runWithoutTTY(ctx, prompt, args, promptPreview)
}

func (c *Client) runWithScript(ctx context.Context, prompt string, args []string, promptPreview string) (string, error) {
	scriptPath, err := exec.LookPath("script")
	if err != nil {
		if c.logger != nil {
			c.logger.Errorf("script command not found: %v", err)
		}
		return "", fmt.Errorf("script command not found; install util-linux or bsdutils")
	}

	scriptArgs := buildScriptArgs(c.command, args)
	cmd := exec.CommandContext(ctx, scriptPath, scriptArgs...)
	cmd.Dir = c.workdir
	c.applyEnv(cmd)

	if c.promptMode == "stdin" {
		cmd.Stdin = strings.NewReader(prompt + "\n")
	}

	return c.runCommand(ctx, cmd, promptPreview)
}

func (c *Client) runWithoutTTY(ctx context.Context, prompt string, args []string, promptPreview string) (string, error) {
	cmd := exec.CommandContext(ctx, c.command, args...)
	cmd.Dir = c.workdir
	c.applyEnv(cmd)

	if c.promptMode == "stdin" {
		cmd.Stdin = strings.NewReader(prompt)
	}

	return c.runCommand(ctx, cmd, promptPreview)
}

func (c *Client) runCommand(ctx context.Context, cmd *exec.Cmd, promptPreview string) (string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		if c.logger != nil {
			c.logger.Errorf("codex start failed: %v", err)
		}
		return "", fmt.Errorf("codex error: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Wait()
	}()

	started := time.Now()
	var ticker *time.Ticker
	if c.progress > 0 {
		ticker = time.NewTicker(c.progress)
		defer ticker.Stop()
	}

	var err error
	for {
		select {
		case err = <-errCh:
			goto done
		case <-ticker.C:
			if c.logger != nil {
				c.logger.Warnf("codex still running: elapsed=%s prompt=%q", time.Since(started).Truncate(time.Second), promptPreview)
			}
		}
	}

done:
	if ctx.Err() == context.DeadlineExceeded {
		if c.logger != nil {
			c.logger.Errorf("codex timeout after %s", c.timeout)
		}
		return "", fmt.Errorf("codex timeout after %s", c.timeout)
	}

	output := strings.TrimSpace(stdout.String())
	errOutput := strings.TrimSpace(stderr.String())

	if err != nil {
		if c.logger != nil {
			if output != "" {
				c.logger.Errorf("codex stdout: %s", output)
			}
			if errOutput != "" {
				c.logger.Errorf("codex stderr: %s", errOutput)
			}
			c.logger.Errorf("codex exit error: %v", err)
		}
		return "", fmt.Errorf("codex failed; check logs")
	}

	if output == "" && errOutput != "" {
		return errOutput, nil
	}
	return output, nil
}

func (c *Client) applyEnv(cmd *exec.Cmd) {
	env := os.Environ()
	if c.disableCPR {
		env = append(env, "PROMPT_TOOLKIT_NO_CPR=1")
	}
	cmd.Env = env
}

func buildScriptArgs(command string, args []string) []string {
	if runtime.GOOS == "darwin" {
		out := []string{"-q", "/dev/null", command}
		return append(out, args...)
	}

	cmdline := shellJoin(append([]string{command}, args...))
	return []string{"-q", "-c", cmdline, "/dev/null"}
}

func shellJoin(args []string) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(input string) string {
	if input == "" {
		return "''"
	}
	needsQuote := false
	for _, r := range input {
		if unicode.IsSpace(r) || strings.ContainsRune("'\"`$|&;<>*?()[]{}!\\", r) {
			needsQuote = true
			break
		}
	}
	if !needsQuote {
		return input
	}
	return "'" + strings.ReplaceAll(input, "'", "'\\''") + "'"
}

func replacePromptPlaceholder(args []string, prompt string) ([]string, bool) {
	used := false
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.Contains(arg, "{prompt}") {
			used = true
			out = append(out, strings.ReplaceAll(arg, "{prompt}", prompt))
			continue
		}
		out = append(out, arg)
	}
	return out, used
}

func buildCommandError(output string, errOutput string, err error) error {
	if errOutput != "" {
		return fmt.Errorf("codex error: %s", errOutput)
	}
	if output != "" {
		return fmt.Errorf("codex error: %s", output)
	}
	return fmt.Errorf("codex error: %v", err)
}

func isTTYError(err error) bool {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "stdin is not a terminal") {
		return true
	}
	if strings.Contains(message, "cursor position could not be read") {
		return true
	}
	return false
}

func truncatePrompt(text string, limit int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}
