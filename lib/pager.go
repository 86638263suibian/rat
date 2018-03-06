package rat

import (
	"fmt"
	"io"
	"strings"
)

type Pager interface {
	Widget
	Window
	AddEventHandler(keyStr string, handler EventHandler)
	Reload()
}

type pager struct {
	title         string
	modes         []Mode
	ctx           Context
	buffer        Buffer
	eventHandlers HandlerRegistry
	*pagerLayout
	Window
}

func newPager(title string, modeNames string, ctx Context) *pager {
	p := &pager{}

	p.title = title
	p.ctx = ctx
	p.eventHandlers = NewHandlerRegistry()
	p.pagerLayout = &pagerLayout{}

	p.Window = NewWindow(
		func() int { return p.GetContentBox().Height() },
		func() int { return p.buffer.NumLines() },
	)

	splitModeNames := strings.Split(modeNames, ",")
	p.modes = make([]Mode, 0, len(splitModeNames))

	for _, modeName := range splitModeNames {
		if mode, ok := modes[modeName]; ok {
			p.modes = append(p.modes, mode)
		}
	}

	return p
}

func (p *pager) AddEventHandler(keyStr string, handler EventHandler) {
	p.eventHandlers.Add(KeySequenceFromString(keyStr), handler)
}

func (p *pager) Stop() {
	p.buffer.Close()
}

func (p *pager) Destroy() {
	p.Stop()
}

func (p *pager) HandleEvent(ks []keyEvent) bool {
	p.buffer.Lock()
	defer p.buffer.Unlock()

	ctx := NewContextFromAnnotations(p.buffer.AnnotationsForLine(p.Window.GetCursor()))

	if handler := p.eventHandlers.FindCtx(ks, ctx); handler != nil {
		handler.Call(ctx)
		return true
	}

	return false
}

func (p *pager) Render() {
	p.buffer.Lock()
	defer p.buffer.Unlock()

	p.drawHeader(
		p.title,
		fmt.Sprintf("%d %d/%d", p.buffer.NumAnnotations(), p.Window.GetCursor()+1, p.buffer.NumLines()),
	)

	p.drawContent(
		p.Window.GetCursor()-p.Window.GetScroll(),
		p.buffer.StyledLines(p.Window.GetScroll(), p.GetContentBox().Height()),
	)
}

func (p *pager) Reload() {
}

func NewReadPager(rd io.Reader, title string, modeNames string, ctx Context) Pager {
	p := newPager(title, modeNames, ctx)

	for _, mode := range p.modes {
		mode.AddEventHandlers(ctx)(p)
	}

	p.buffer = NewBuffer(rd, p.annotators)

	return p
}

type cmdPager struct {
	*pager
	cmd     string
	command ShellCommand
}

func NewCmdPager(modeNames string, cmd string, ctx Context) Pager {
	cp := &cmdPager{}

	cp.cmd = cmd
	cp.pager = newPager(cmd, modeNames, ctx)

	for _, mode := range cp.modes {
		mode.AddEventHandlers(ctx)(cp)
	}

	cp.RunCommand()

	return cp
}

func (cp *cmdPager) Stop() {
	cp.command.Close()
	cp.pager.Stop()
}

func (cp *cmdPager) Reload() {
	cp.Stop()
	cp.RunCommand()
}

func (cp *cmdPager) RunCommand() {
	var err error

	if cp.command, err = NewShellCommand(cp.cmd, cp.ctx); err != nil {
		panic(err)
	}

	cp.buffer = NewBuffer(cp.command, cp.pager.annotators)
}
