package blocks

import "strings"

const executorGuardLimit = 64

type Host interface {
	ResetInputs()
	OnDeactivate()
	EvalCondition(condition string, params map[string]any) bool
	ExecAction(blockType string, params map[string]any, delta float64, state *ActionState) bool
}

type ActionState struct {
	HasHeading    bool
	Heading       float64
	Remaining     float64
	HasStepTime   bool
	StepTime      float64
	HasTurnAmount bool
	TurnAmount    float64
}

func (s *ActionState) Reset() {
	*s = ActionState{}
}

type stackFrame struct {
	list    []Block
	index   int
	matched bool
}

type runtimeScript struct {
	condition  string
	condParams map[string]any
	body       []Block
	frames     []stackFrame
	state      ActionState
	active     bool
	once       bool
	done       bool
}

type Executor struct {
	host    Host
	scripts []runtimeScript
}

func NewExecutor(host Host, scripts []Script) *Executor {
	executor := &Executor{host: host}
	for _, script := range scripts {
		executor.collect(script.Blocks)
	}
	return executor
}

func (e *Executor) collect(list []Block) {
	current := -1
	for _, block := range list {
		if strings.HasPrefix(block.Type, "when_") {
			condition := block.Type[len("when_"):]
			e.scripts = append(e.scripts, runtimeScript{
				condition:  condition,
				condParams: block.Params,
				body:       append([]Block(nil), block.Children...),
				once:       condition == "start",
			})
			current = len(e.scripts) - 1
			continue
		}
		if current >= 0 {
			e.scripts[current].body = append(e.scripts[current].body, block)
		}
	}
}

func (e *Executor) Process(delta float64) {
	e.host.ResetInputs()
	for index := range e.scripts {
		e.runScript(&e.scripts[index], delta)
	}
}

func (e *Executor) runScript(script *runtimeScript, delta float64) {
	if script.done {
		return
	}
	if len(script.body) == 0 {
		if script.once {
			script.done = true
		}
		return
	}
	if !(script.once || e.host.EvalCondition(script.condition, script.condParams)) {
		if script.active {
			e.host.OnDeactivate()
		}
		script.active = false
		script.frames = script.frames[:0]
		script.state.Reset()
		return
	}
	if !script.active {
		script.active = true
		script.frames = append(script.frames[:0], stackFrame{list: script.body})
		script.state.Reset()
	}

	for guard := 0; guard < executorGuardLimit; guard++ {
		if len(script.frames) == 0 {
			if script.once {
				script.done = true
				return
			}
			script.frames = append(script.frames[:0], stackFrame{list: script.body})
			script.state.Reset()
			continue
		}

		frame := &script.frames[len(script.frames)-1]
		if frame.index >= len(frame.list) {
			script.frames = script.frames[:len(script.frames)-1]
			if len(script.frames) > 0 {
				script.frames[len(script.frames)-1].index++
			}
			script.state.Reset()
			continue
		}

		block := &frame.list[frame.index]
		if block.Type == "else" {
			runElse := !frame.matched
			frame.matched = false
			if runElse {
				script.frames = append(script.frames, stackFrame{list: block.Children})
				script.state.Reset()
			} else {
				frame.index++
			}
			continue
		}

		if strings.HasPrefix(block.Type, "when_") || strings.HasPrefix(block.Type, "if_") {
			separator := strings.Index(block.Type, "_")
			matched := e.host.EvalCondition(block.Type[separator+1:], block.Params)
			frame.matched = matched
			if matched {
				script.frames = append(script.frames, stackFrame{list: block.Children})
				script.state.Reset()
			} else {
				frame.index++
			}
			continue
		}

		if e.host.ExecAction(block.Type, block.Params, delta, &script.state) {
			frame.index++
			script.state.Reset()
			continue
		}
		return
	}
}
