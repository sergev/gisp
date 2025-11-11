package parser

import (
	"fmt"

	"github.com/sergev/gisp/lang"
)

type builder struct {
	gensymCounter int
}

func (b *builder) gensym(prefix string) string {
	b.gensymCounter++
	return fmt.Sprintf("__gisp_%s_%d", prefix, b.gensymCounter)
}

func (b *builder) symbol(name string) lang.Value {
	return lang.SymbolValue(name)
}

func (b *builder) list(values ...lang.Value) lang.Value {
	return lang.List(values...)
}

func (b *builder) begin(forms []lang.Value) lang.Value {
	switch len(forms) {
	case 0:
		return lang.EmptyList
	case 1:
		return forms[0]
	default:
		all := make([]lang.Value, 0, len(forms)+1)
		all = append(all, lang.SymbolValue("begin"))
		all = append(all, forms...)
		return lang.List(all...)
	}
}

func (b *builder) lambda(params []string, body lang.Value) lang.Value {
	paramList := lang.EmptyList
	for i := len(params) - 1; i >= 0; i-- {
		paramList = lang.PairValue(b.symbol(params[i]), paramList)
	}
	return b.list(
		b.symbol("lambda"),
		paramList,
		body,
	)
}

func (b *builder) let(bindings []binding, body lang.Value) lang.Value {
	bindList := lang.EmptyList
	for i := len(bindings) - 1; i >= 0; i-- {
		bind := bindings[i]
		pair := lang.List(b.symbol(bind.name), bind.value)
		bindList = lang.PairValue(pair, bindList)
	}
	return b.list(
		b.symbol("let"),
		bindList,
		body,
	)
}

type binding struct {
	name  string
	value lang.Value
}

func (b *builder) quoteSymbol(name string) lang.Value {
	return b.list(
		b.symbol("quote"),
		b.symbol(name),
	)
}
