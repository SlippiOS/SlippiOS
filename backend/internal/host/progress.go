package host

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

type Progress struct {
	total   int64
	done    int64
	label   string
	out     io.Writer
	tty     bool
	last    int
	started bool
	mu      sync.Mutex
}

func newProgress(total int64, label string, out *os.File) *Progress {
	return NewProgress(total, label, out)
}

func NewProgress(total int64, label string, out io.Writer) *Progress {
	tty := false
	if file, ok := out.(*os.File); ok {
		tty = IsTerminal(file)
	}
	return &Progress{total: total, label: label, out: out, tty: tty}
}

func (p *Progress) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done += int64(len(data))
	p.render()
	return len(data), nil
}

func (p *Progress) Add(count int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done += count
	p.render()
}

func (p *Progress) render() {
	percent := 0
	if p.total > 0 {
		percent = int(p.done * 100 / p.total)
		if percent > 100 {
			percent = 100
		}
	}
	if p.tty {
		width := 40
		filled := percent * width / 100
		bar := strings.Repeat("#", filled) + strings.Repeat(" ", width-filled)
		if p.total > 0 {
			fmt.Fprintf(p.out, "\r[%s] %3d%%  %s / %s", bar, percent, HumanSize(p.done), HumanSize(p.total))
		} else {
			fmt.Fprintf(p.out, "\r[%s] %3d%%  %s", bar, percent, HumanSize(p.done))
		}
		p.started = true
		return
	}
	step := percent / 10 * 10
	if step > p.last || (!p.started && percent > 0) {
		p.last = step
		p.started = true
		if p.total > 0 {
			fmt.Fprintf(p.out, "%s: %d%% (%s / %s)\n", p.label, percent, HumanSize(p.done), HumanSize(p.total))
		} else {
			fmt.Fprintf(p.out, "%s: %s\n", p.label, HumanSize(p.done))
		}
	}
}

func (p *Progress) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.tty && p.started {
		fmt.Fprintln(p.out)
	}
	p.started = false
}
