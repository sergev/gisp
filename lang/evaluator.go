package lang

import "fmt"

// Evaluator executes Scheme-like programs.
type Evaluator struct {
	Global *Env
}

// NewEvaluator constructs an evaluator rooted at a new global environment.
func NewEvaluator() *Evaluator {
	global := NewEnv(nil)
	return &Evaluator{Global: global}
}

// Eval evaluates a single expression within the provided environment.
func (ev *Evaluator) Eval(expr Value, env *Env) (Value, error) {
	if env == nil {
		env = ev.Global
	}
	state := &evalState{
		expr: expr,
		env:  env,
	}
	return ev.run(state)
}

// Apply invokes a procedure with arguments.
func (ev *Evaluator) Apply(proc Value, args []Value) (Value, error) {
	state := &evalState{}
	if err := ev.invokeProcedure(state, proc, args); err != nil {
		return Value{}, err
	}
	return ev.run(state)
}

func (ev *Evaluator) run(state *evalState) (Value, error) {
	for {
		if state.returning {
			if len(state.cont) == 0 {
				return state.value, nil
			}
			frame := state.pop()
			if err := frame.apply(ev, state.value, state); err != nil {
				return Value{}, err
			}
			continue
		}
		if err := ev.evaluateCurrent(state); err != nil {
			return Value{}, err
		}
	}
}

// EvalAll evaluates a sequence of expressions.
func (ev *Evaluator) EvalAll(exprs []Value, env *Env) (Value, error) {
	result := EmptyList
	for _, expr := range exprs {
		val, err := ev.Eval(expr, env)
		if err != nil {
			return Value{}, err
		}
		result = val
	}
	return result, nil
}

type evalState struct {
	expr      Value
	env       *Env
	cont      []frame
	value     Value
	returning bool
}

func (st *evalState) push(f frame) {
	st.cont = append(st.cont, f)
}

func (st *evalState) pop() frame {
	l := len(st.cont)
	if l == 0 {
		return nil
	}
	f := st.cont[l-1]
	st.cont = st.cont[:l-1]
	return f
}

func (st *evalState) setExpr(expr Value, env *Env) {
	st.expr = expr
	if env != nil {
		st.env = env
	}
	st.returning = false
}

type frame interface {
	apply(ev *Evaluator, val Value, state *evalState) error
	clone() frame
}

func (ev *Evaluator) evaluateCurrent(state *evalState) error {
	switch state.expr.Type {
	case TypeSymbol:
		val, err := state.env.Get(state.expr.Sym())
		if err != nil {
			return err
		}
		state.value = val
		state.returning = true
	case TypePair:
		return ev.evaluatePair(state)
	default:
		state.value = state.expr
		state.returning = true
	}
	return nil
}

func (ev *Evaluator) evaluatePair(state *evalState) error {
	list := state.expr
	pair := list.Pair()
	if pair == nil {
		return fmt.Errorf("expected pair value")
	}
	head := pair.First

	if head.Type == TypeSymbol {
		switch head.Sym() {
		case "quote":
			return ev.evalQuote(pair.Rest, state)
		case "if":
			return ev.evalIf(pair.Rest, state)
		case "begin":
			return ev.evalBegin(pair.Rest, state)
		case "lambda":
			return ev.evalLambda(pair.Rest, state)
		case "define":
			return ev.evalDefine(pair.Rest, state)
		case "define-macro":
			return ev.evalDefineMacro(pair.Rest, state)
		case "set!":
			return ev.evalSet(pair.Rest, state)
		case "let":
			return ev.evalLet(pair.Rest, state)
		case "quasiquote":
			return ev.evalQuasiQuote(pair.Rest, state)
		case "call/cc":
			return ev.evalCallCC(pair.Rest, state)
		case "cond":
			return ev.evalCond(pair.Rest, state)
		}
	}

	if head.Type == TypeSymbol {
		if macroVal, err := state.env.Get(head.Sym()); err == nil && macroVal.Type == TypeMacro {
			expanded, err := ev.expandMacro(macroVal.Macro(), pair.Rest, state.env)
			if err != nil {
				return err
			}
			state.setExpr(expanded, state.env)
			return nil
		}
	}

	frame := &callFrame{
		env:       state.env,
		remaining: pair.Rest,
	}
	state.push(frame)
	state.setExpr(pair.First, state.env)
	return nil
}

func (ev *Evaluator) evalQuote(args Value, state *evalState) error {
	exprs, err := ToSlice(args)
	if err != nil {
		return err
	}
	if len(exprs) != 1 {
		return fmt.Errorf("quote expects 1 argument")
	}
	state.value = exprs[0]
	state.returning = true
	return nil
}

type ifFrame struct {
	consequent Value
	alternate  Value
	env        *Env
}

func (f *ifFrame) apply(ev *Evaluator, val Value, state *evalState) error {
	var next Value
	if IsTruthy(val) {
		next = f.consequent
	} else {
		next = f.alternate
	}
	state.setExpr(next, f.env)
	return nil
}

func (f *ifFrame) clone() frame {
	return &ifFrame{
		consequent: f.consequent,
		alternate:  f.alternate,
		env:        f.env,
	}
}

func (ev *Evaluator) evalCond(args Value, state *evalState) error {
	clauses, err := ToSlice(args)
	if err != nil {
		return fmt.Errorf("cond expects a list of clauses: %w", err)
	}
	return ev.runCondClauses(clauses, state.env, state)
}

type condFrame struct {
	remaining []Value
	body      Value
	env       *Env
}

func (f *condFrame) apply(ev *Evaluator, val Value, state *evalState) error {
	if IsTruthy(val) {
		state.setExpr(f.body, f.env)
		return nil
	}
	return ev.runCondClauses(f.remaining, f.env, state)
}

func (f *condFrame) clone() frame {
	var remainingCopy []Value
	if f.remaining != nil {
		remainingCopy = append([]Value(nil), f.remaining...)
	}
	return &condFrame{
		remaining: remainingCopy,
		body:      f.body,
		env:       f.env,
	}
}

func (ev *Evaluator) runCondClauses(clauses []Value, env *Env, state *evalState) error {
	if len(clauses) == 0 {
		state.value = EmptyList
		state.returning = true
		return nil
	}
	clause := clauses[0]
	items, err := ToSlice(clause)
	if err != nil {
		return fmt.Errorf("cond clause must be a list: %w", err)
	}
	if len(items) != 2 {
		return fmt.Errorf("cond clause must have predicate and result expression")
	}
	predicate := items[0]
	body := items[1]
	if predicate.Type == TypeSymbol && predicate.Sym() == "else" {
		if len(clauses) != 1 {
			return fmt.Errorf("cond else clause must be last")
		}
		state.setExpr(body, env)
		return nil
	}
	frame := &condFrame{
		remaining: clauses[1:],
		body:      body,
		env:       env,
	}
	state.push(frame)
	state.setExpr(predicate, env)
	return nil
}

func (ev *Evaluator) evalIf(args Value, state *evalState) error {
	parts, err := ToSlice(args)
	if err != nil {
		return err
	}
	if len(parts) < 2 || len(parts) > 3 {
		return fmt.Errorf("if expects 2 or 3 arguments")
	}
	alt := EmptyList
	if len(parts) == 3 {
		alt = parts[2]
	}
	state.push(&ifFrame{
		consequent: parts[1],
		alternate:  alt,
		env:        state.env,
	})
	state.setExpr(parts[0], state.env)
	return nil
}

type beginFrame struct {
	exprs []Value
	env   *Env
}

func (f *beginFrame) apply(ev *Evaluator, val Value, state *evalState) error {
	if len(f.exprs) == 0 {
		state.value = val
		state.returning = true
		return nil
	}
	next := f.exprs[0]
	rest := f.exprs[1:]
	if len(rest) > 0 {
		state.push(&beginFrame{exprs: rest, env: f.env})
	}
	state.setExpr(next, f.env)
	return nil
}

func (f *beginFrame) clone() frame {
	cp := make([]Value, len(f.exprs))
	copy(cp, f.exprs)
	return &beginFrame{
		exprs: cp,
		env:   f.env,
	}
}

func (ev *Evaluator) evalBegin(args Value, state *evalState) error {
	exprs, err := ToSlice(args)
	if err != nil {
		return err
	}
	if len(exprs) == 0 {
		state.value = EmptyList
		state.returning = true
		return nil
	}
	first := exprs[0]
	rest := exprs[1:]
	if len(rest) > 0 {
		state.push(&beginFrame{exprs: rest, env: state.env})
	}
	state.setExpr(first, state.env)
	return nil
}

func (ev *Evaluator) evalLambda(args Value, state *evalState) error {
	parts, err := ToSlice(args)
	if err != nil {
		return err
	}
	if len(parts) < 2 {
		return fmt.Errorf("lambda expects parameters and body")
	}
	paramList := parts[0]
	params, rest, err := parseParams(paramList)
	if err != nil {
		return err
	}
	body := parts[1:]
	closure := ClosureValue(params, rest, body, state.env)
	state.value = closure
	state.returning = true
	return nil
}

func (ev *Evaluator) evalDefine(args Value, state *evalState) error {
	parts, err := ToSlice(args)
	if err != nil {
		return err
	}
	if len(parts) < 2 {
		return fmt.Errorf("define expects a name and value")
	}
	target := parts[0]
	body := parts[1:]

	if target.Type == TypeSymbol {
		if len(body) != 1 {
			return fmt.Errorf("define expects a single value expression")
		}
		state.push(&defineFrame{name: target.Sym(), env: state.env})
		state.setExpr(body[0], state.env)
		return nil
	}

	if target.Type == TypePair {
		targetPair := target.Pair()
		if targetPair == nil {
			return fmt.Errorf("invalid function definition target")
		}
		nameVal := targetPair.First
		if nameVal.Type != TypeSymbol {
			return fmt.Errorf("function name in define must be a symbol")
		}
		paramsVal := targetPair.Rest
		params, rest, err := parseParams(paramsVal)
		if err != nil {
			return err
		}
		lambda := ClosureValue(params, rest, body, state.env)
		state.env.Define(nameVal.Sym(), lambda)
		state.value = lambda
		state.returning = true
		return nil
	}

	return fmt.Errorf("invalid define target")
}

type defineFrame struct {
	name string
	env  *Env
}

func (f *defineFrame) apply(ev *Evaluator, val Value, state *evalState) error {
	f.env.Define(f.name, val)
	state.value = val
	state.returning = true
	return nil
}

func (f *defineFrame) clone() frame {
	return &defineFrame{name: f.name, env: f.env}
}

func (ev *Evaluator) evalDefineMacro(args Value, state *evalState) error {
	parts, err := ToSlice(args)
	if err != nil {
		return err
	}
	if len(parts) < 2 || parts[0].Type != TypePair {
		return fmt.Errorf("define-macro expects (name params) head")
	}
	head := parts[0]
	body := parts[1:]
	headPair := head.Pair()
	if headPair == nil {
		return fmt.Errorf("macro head must be a pair")
	}
	nameVal := headPair.First
	if nameVal.Type != TypeSymbol {
		return fmt.Errorf("macro name must be a symbol")
	}
	params, rest, err := parseParams(headPair.Rest)
	if err != nil {
		return err
	}
	macro := MacroValue(params, rest, body, state.env)
	state.env.Define(nameVal.Sym(), macro)
	state.value = macro
	state.returning = true
	return nil
}

func (ev *Evaluator) evalSet(args Value, state *evalState) error {
	parts, err := ToSlice(args)
	if err != nil {
		return err
	}
	if len(parts) != 2 {
		return fmt.Errorf("set! expects a name and value")
	}
	nameVal := parts[0]
	if nameVal.Type != TypeSymbol {
		return fmt.Errorf("set! target must be a symbol")
	}
	state.push(&setFrame{name: nameVal.Sym(), env: state.env})
	state.setExpr(parts[1], state.env)
	return nil
}

type setFrame struct {
	name string
	env  *Env
}

func (f *setFrame) apply(ev *Evaluator, val Value, state *evalState) error {
	if err := f.env.Set(f.name, val); err != nil {
		return err
	}
	state.value = val
	state.returning = true
	return nil
}

func (f *setFrame) clone() frame {
	return &setFrame{name: f.name, env: f.env}
}

func (ev *Evaluator) evalLet(args Value, state *evalState) error {
	parts, err := ToSlice(args)
	if err != nil {
		return err
	}
	if len(parts) < 2 {
		return fmt.Errorf("let expects bindings and body")
	}
	bindings := parts[0]
	bodyStart := 1
	var letName string
	if bindings.Type == TypeSymbol {
		letName = bindings.Sym()
		if len(parts) < 3 {
			return fmt.Errorf("named let expects bindings and body")
		}
		bindings = parts[1]
		bodyStart = 2
	}
	body := parts[bodyStart:]
	names := []Value{}
	values := []Value{}

	iter := bindings
	for iter.Type != TypeEmpty {
		if iter.Type != TypePair {
			return fmt.Errorf("invalid binding list")
		}
		iterPair := iter.Pair()
		if iterPair == nil {
			return fmt.Errorf("invalid binding list")
		}
		bind := iterPair.First
		if bind.Type != TypePair {
			return fmt.Errorf("binding must be a list")
		}
		bPair := bind.Pair()
		if bPair == nil {
			return fmt.Errorf("binding must be a pair")
		}
		name := bPair.First
		if name.Type != TypeSymbol {
			return fmt.Errorf("binding name must be a symbol")
		}
		valueList := bPair.Rest
		valueSlice, err := ToSlice(valueList)
		if err != nil || len(valueSlice) != 1 {
			return fmt.Errorf("binding must have exactly one value")
		}
		names = append(names, name)
		values = append(values, valueSlice[0])
		iter = iterPair.Rest
	}
	paramNames := make([]Value, len(names))
	copy(paramNames, names)
	lambdaParams := EmptyList
	for i := len(paramNames) - 1; i >= 0; i-- {
		lambdaParams = PairValue(paramNames[i], lambdaParams)
	}
	lambdaList := append([]Value{SymbolValue("lambda"), lambdaParams}, body...)
	lambdaExpr := List(lambdaList...)
	if letName != "" {
		binding := List(SymbolValue(letName), EmptyList)
		bindingList := List(binding)
		setExpr := List(SymbolValue("set!"), SymbolValue(letName), lambdaExpr)
		callArgs := append([]Value{SymbolValue(letName)}, values...)
		callExpr := List(callArgs...)
		letParts := append([]Value{SymbolValue("let"), bindingList}, []Value{setExpr, callExpr}...)
		state.setExpr(List(letParts...), state.env)
		return nil
	}
	callList := []Value{lambdaExpr}
	callList = append(callList, values...)
	state.setExpr(List(callList...), state.env)
	return nil
}

func (ev *Evaluator) evalQuasiQuote(args Value, state *evalState) error {
	exprs, err := ToSlice(args)
	if err != nil {
		return err
	}
	if len(exprs) != 1 {
		return fmt.Errorf("quasiquote expects 1 argument")
	}
	expanded, err := expandQuasiQuote(exprs[0], 1)
	if err != nil {
		return err
	}
	state.setExpr(expanded, state.env)
	return nil
}

func (ev *Evaluator) evalCallCC(args Value, state *evalState) error {
	exprs, err := ToSlice(args)
	if err != nil {
		return err
	}
	if len(exprs) != 1 {
		return fmt.Errorf("call/cc expects single argument")
	}
	frame := &callCCFrame{
		env:   state.env,
		stack: cloneFrames(state.cont),
	}
	state.push(frame)
	state.setExpr(exprs[0], state.env)
	return nil
}

type callCCFrame struct {
	env   *Env
	stack []frame
}

func (f *callCCFrame) apply(ev *Evaluator, val Value, state *evalState) error {
	contVal := ContinuationValue(cloneFrames(f.stack), f.env, ev)
	return ev.invokeProcedure(state, val, []Value{contVal})
}

func (f *callCCFrame) clone() frame {
	return &callCCFrame{
		env:   f.env,
		stack: cloneFrames(f.stack),
	}
}

func (ev *Evaluator) expandMacro(m *Macro, args Value, env *Env) (Value, error) {
	argValues, err := listToSliceRaw(args)
	if err != nil {
		return Value{}, err
	}
	callEnv := NewEnv(m.Env)
	if err := bindParameters(callEnv, m.Params, m.Rest, argValues); err != nil {
		return Value{}, err
	}
	var result Value = EmptyList
	for _, expr := range m.Body {
		val, err := ev.Eval(expr, callEnv)
		if err != nil {
			return Value{}, err
		}
		result = val
	}
	return result, nil
}

func (ev *Evaluator) invokeProcedure(state *evalState, operator Value, args []Value) error {
	switch operator.Type {
	case TypePrimitive:
		fn := operator.Primitive()
		if fn == nil {
			return fmt.Errorf("invalid primitive")
		}
		val, err := fn(ev, args)
		if err != nil {
			return err
		}
		state.value = val
		state.returning = true
	case TypeClosure:
		closure := operator.Closure()
		if closure == nil {
			return fmt.Errorf("invalid closure")
		}
		newEnv := NewEnv(closure.Env)
		if err := bindParameters(newEnv, closure.Params, closure.Rest, args); err != nil {
			return err
		}
		body := closure.Body
		if len(body) == 0 {
			state.value = EmptyList
			state.returning = true
			return nil
		}
		first := body[0]
		rest := body[1:]
		if len(rest) > 0 {
			state.push(&beginFrame{exprs: rest, env: newEnv})
		}
		state.setExpr(first, newEnv)
	case TypeContinuation:
		cont := operator.Continuation()
		if cont == nil || cont.Eval == nil {
			return fmt.Errorf("invalid continuation")
		}
		var arg Value = EmptyList
		if len(args) > 0 {
			arg = args[0]
		}
		state.cont = cloneFrames(cont.Frames)
		state.env = cont.Env
		state.value = arg
		state.returning = true
	default:
		return fmt.Errorf("attempt to call non-function: %s", operator.String())
	}
	return nil
}

type callFrame struct {
	env          *Env
	operator     Value
	remaining    Value
	args         []Value
	operatorDone bool
}

func (f *callFrame) apply(ev *Evaluator, val Value, state *evalState) error {
	if !f.operatorDone {
		f.operator = val
		f.operatorDone = true
	} else {
		f.args = append(f.args, val)
	}

	if f.remaining.Type == TypeEmpty {
		return ev.invokeProcedure(state, f.operator, f.args)
	}

	if f.remaining.Type != TypePair {
		return fmt.Errorf("malformed argument list")
	}
	remPair := f.remaining.Pair()
	if remPair == nil {
		return fmt.Errorf("malformed argument list")
	}
	next := remPair.First
	f.remaining = remPair.Rest
	state.push(f)
	state.setExpr(next, f.env)
	return nil
}

func (f *callFrame) clone() frame {
	argsCopy := make([]Value, len(f.args))
	copy(argsCopy, f.args)
	return &callFrame{
		env:          f.env,
		operator:     f.operator,
		remaining:    f.remaining,
		args:         argsCopy,
		operatorDone: f.operatorDone,
	}
}

func parseParams(val Value) ([]string, string, error) {
	var params []string
	var rest string
	for val.Type != TypeEmpty {
		if val.Type == TypeSymbol {
			if rest != "" {
				return nil, "", fmt.Errorf("multiple rest parameters")
			}
			rest = val.Sym()
			break
		}
		if val.Type != TypePair {
			return nil, "", fmt.Errorf("invalid parameter list")
		}
		p := val.Pair()
		if p == nil {
			return nil, "", fmt.Errorf("invalid parameter list")
		}
		name := p.First
		if name.Type != TypeSymbol {
			return nil, "", fmt.Errorf("parameter must be a symbol")
		}
		params = append(params, name.Sym())
		val = p.Rest
	}
	return params, rest, nil
}

func bindParameters(env *Env, params []string, rest string, args []Value) error {
	if len(args) < len(params) {
		return fmt.Errorf("expected at least %d arguments, got %d", len(params), len(args))
	}
	for i, name := range params {
		env.Define(name, args[i])
	}
	if rest != "" {
		env.Define(rest, listFromArgs(args[len(params):]))
	} else if len(args) != len(params) {
		return fmt.Errorf("expected exactly %d arguments, got %d", len(params), len(args))
	}
	return nil
}

func listFromArgs(args []Value) Value {
	if len(args) == 0 {
		return EmptyList
	}
	return List(args...)
}

func expandQuasiQuote(expr Value, depth int) (Value, error) {
	switch expr.Type {
	case TypePair:
		p := expr.Pair()
		if p == nil {
			return Value{}, fmt.Errorf("expected pair")
		}
		head := p.First
		tail := p.Rest
		if tagged, ok, err := taggedForm(head, "unquote"); err != nil {
			return Value{}, err
		} else if ok {
			if depth == 1 {
				return tagged, nil
			}
			sub, err := expandQuasiQuote(tagged, depth-1)
			if err != nil {
				return Value{}, err
			}
			tailExpanded, err := expandQuasiQuote(tail, depth)
			if err != nil {
				return Value{}, err
			}
			return List(SymbolValue("cons"), List(SymbolValue("quote"), SymbolValue("unquote")), List(SymbolValue("cons"), sub, tailExpanded)), nil
		}
		if tagged, ok, err := taggedForm(head, "unquote-splicing"); err != nil {
			return Value{}, err
		} else if ok {
			if depth == 1 {
				tailExpanded, err := expandQuasiQuote(tail, depth)
				if err != nil {
					return Value{}, err
				}
				return List(SymbolValue("append"), tagged, tailExpanded), nil
			}
			sub, err := expandQuasiQuote(tagged, depth-1)
			if err != nil {
				return Value{}, err
			}
			tailExpanded, err := expandQuasiQuote(tail, depth)
			if err != nil {
				return Value{}, err
			}
			return List(SymbolValue("cons"), List(SymbolValue("quote"), SymbolValue("unquote-splicing")), List(SymbolValue("cons"), sub, tailExpanded)), nil
		}
		if tagged, ok, err := taggedForm(head, "quasiquote"); err != nil {
			return Value{}, err
		} else if ok {
			sub, err := expandQuasiQuote(tagged, depth+1)
			if err != nil {
				return Value{}, err
			}
			tailExpanded, err := expandQuasiQuote(tail, depth)
			if err != nil {
				return Value{}, err
			}
			return List(SymbolValue("cons"), List(SymbolValue("quote"), SymbolValue("quasiquote")), List(SymbolValue("cons"), sub, tailExpanded)), nil
		}
		headExpanded, err := expandQuasiQuote(head, depth)
		if err != nil {
			return Value{}, err
		}
		tailExpanded, err := expandQuasiQuote(tail, depth)
		if err != nil {
			return Value{}, err
		}
		return List(SymbolValue("cons"), headExpanded, tailExpanded), nil
	case TypeSymbol:
		return List(SymbolValue("quote"), expr), nil
	case TypeEmpty:
		return List(SymbolValue("quote"), EmptyList), nil
	default:
		return expr, nil
	}
}

func isSymbolNamed(v Value, name string) bool {
	return v.Type == TypeSymbol && v.Sym() == name
}

func taggedForm(v Value, tag string) (Value, bool, error) {
	if v.Type != TypePair {
		return Value{}, false, nil
	}
	p := v.Pair()
	if p == nil {
		return Value{}, false, nil
	}
	if !isSymbolNamed(p.First, tag) {
		return Value{}, false, nil
	}
	args, err := listToSliceRaw(p.Rest)
	if err != nil {
		return Value{}, false, err
	}
	if len(args) != 1 {
		return Value{}, false, fmt.Errorf("%s expects 1 argument", tag)
	}
	return args[0], true, nil
}

func listToSliceRaw(list Value) ([]Value, error) {
	var out []Value
	cur := list
	for cur.Type != TypeEmpty {
		if cur.Type != TypePair {
			return nil, fmt.Errorf("expected proper list")
		}
		p := cur.Pair()
		if p == nil {
			return nil, fmt.Errorf("expected proper list")
		}
		out = append(out, p.First)
		cur = p.Rest
	}
	return out, nil
}

func cloneFrames(frames []frame) []frame {
	if len(frames) == 0 {
		return nil
	}
	out := make([]frame, len(frames))
	for i, fr := range frames {
		out[i] = fr.clone()
	}
	return out
}

// IsTruthy reports whether a value counts as true.
func IsTruthy(v Value) bool {
	return !(v.Type == TypeBool && !v.Bool())
}
