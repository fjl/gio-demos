package main

import (
	"strconv"
)

const (
	opEq calcOp = iota
	opAdd
	opSub
	opMul
	opDiv
	opNop
)

type calcOp int

func (op calcOp) String() string {
	switch op {
	case opNop:
		return "nop"
	case opEq:
		return "="
	case opAdd:
		return "+"
	case opSub:
		return "-"
	case opMul:
		return "*"
	case opDiv:
		return "/"
	default:
		panic("unknown op")
	}
}

// apply computes the operation.
func (op calcOp) apply(x, y float64) float64 {
	switch op {
	case opNop, opEq:
		return y
	case opAdd:
		return x + y
	case opSub:
		return x - y
	case opMul:
		return x * y
	case opDiv:
		return x / y
	default:
		panic("unknown op")
	}
}

type calculator struct {
	input           string
	top             float64
	queued          float64
	lastOp          calcOp
	nextDigitResets bool
}

// digit processes an input digit.
func (c *calculator) digit(in string) bool {
	if c.nextDigitResets {
		c.resetInput()
		c.nextDigitResets = false
	}
	if len(in) > 1 {
		panic("bad digit")
	}
	switch {
	case in[0] == '.':
		for i := range c.input {
			if c.input[i] == '.' {
				return false
			}
		}
		c.input += in
		return true
	case in[0] >= '0' && in[0] <= '9':
		c.input += in
		return c.parse(c.input)
	default:
		return false
	}
}

// resetInput clears the input.
func (c *calculator) resetInput() {
	c.top = 0
	c.input = ""
}

// reset clears the calculator.
func (c *calculator) reset() {
	c.resetInput()
	c.lastOp = opEq
	c.queued = 0
	c.nextDigitResets = false
}

// rubout undoes the last input.
func (c *calculator) rubout() {
	if len(c.input) > 0 {
		c.input = c.input[:len(c.input)-1]
		c.parse(c.input)
	}
}

// parse reads the given input.
func (c *calculator) parse(input string) bool {
	if input == "" {
		c.top = 0
		return true
	}
	num, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return false
	}
	c.top = num
	c.input = input
	return true
}

// percent divides by 100.
func (c *calculator) percent() {
	c.top /= 100
	c.input = ""
}

// flipSign flips the sign of the input between positive and negative.
func (c *calculator) flipSign() {
	c.top *= -1
	c.input = ""
}

// run applies the given operation.
func (c *calculator) run(op calcOp) {
	if op == c.lastOp && c.nextDigitResets {
		return
	}
	c.top = c.lastOp.apply(c.queued, c.top)
	c.input = ""
	c.queued = c.top
	c.lastOp = op
	c.nextDigitResets = true
}

// text gives the current output of the calculator.
func (c *calculator) text() string {
	if len(c.input) > 0 {
		return c.input
	}
	return strconv.FormatFloat(c.top, 'g', 12, 64)
}
