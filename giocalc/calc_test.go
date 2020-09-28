package main

import "testing"

func TestCalcInput(t *testing.T) {
	var c calculator
	// input integer
	c.digit("1")
	c.digit("2")
	c.digit("3")
	check(t, c, "123")
	// redo last digit
	c.rubout()
	c.digit("4")
	check(t, c, "124")
	// decimal point
	c.digit(".")
	check(t, c, "124.")
	c.digit("6")
	c.digit("7")
	check(t, c, "124.67")
	// rubout decimals
	c.rubout()
	check(t, c, "124.6")
	c.rubout()
	check(t, c, "124.")
	c.rubout()
	check(t, c, "124")
}

func TestCalcBadInput(t *testing.T) {
	var c calculator
	c.digit("1")
	c.digit("2")
	c.digit("3")
	check(t, c, "123")
	c.digit("a")
	check(t, c, "123")
	c.digit(".")
	check(t, c, "123.")
	c.digit("2")
	check(t, c, "123.2")
	c.digit(".")
	check(t, c, "123.2")
}

func TestCalcParse(t *testing.T) {
	var c calculator
	c.parse("134.2")
	check(t, c, "134.2")
	c.run(opDiv)
	check(t, c, "134.2")
	c.digit("2")
	check(t, c, "2")
	c.run(opEq)
	check(t, c, "67.1")
}

func TestCalcOpTwice(t *testing.T) {
	var c calculator
	c.parse("1334")
	c.run(opDiv)
	c.run(opDiv)
	check(t, c, "1334")
	c.run(opEq)
	check(t, c, "1")
}

func check(t *testing.T, c calculator, text string) {
	t.Helper()
	if c.text() != text {
		t.Fatalf("wrong text\n  got: %q\n want: %q\nstate: %+v", c.text(), text, c)
	}
}
