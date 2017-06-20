// Copyright 2017 Dyson Simmons. All rights reserved.

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package envvar_test

import (
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	. "github.com/dyson/envvar"
)

func boolString(s string) string {
	if s == "0" {
		return "false"
	}
	return "true"
}

func TestEverything(t *testing.T) {
	ResetForTesting()
	Bool("TEST_BOOL", false)
	Int("TEST_INT", 0)
	Int64("TEST_INT64", 0)
	Uint("TEST_UINT", 0)
	Uint64("TEST_UINT64", 0)
	String("TEST_STRING", "0")
	Float64("TEST_FLOAT64", 0)
	Duration("TEST_DURATION", 0)

	m := make(map[string]*EnvVar)
	desired := "0"
	visitor := func(ev *EnvVar) {
		if len(ev.Name) > 5 && ev.Name[0:5] == "TEST_" {
			m[ev.Name] = ev
			ok := false
			switch {
			case ev.Value.String() == desired:
				ok = true
			case ev.Name == "TEST_BOOL" && ev.Value.String() == boolString(desired):
				ok = true
			case ev.Name == "TEST_DURATION" && ev.Value.String() == desired+"s":
				ok = true
			}
			if !ok {
				t.Error("Visit: bad value", ev.Value.String(), "for", ev.Name)
			}
		}
	}
	VisitAll(visitor)
	if len(m) != 8 {
		t.Error("VisitAll misses some defined envvars")
		for k, v := range m {
			t.Log(k, *v)
		}
	}
	m = make(map[string]*EnvVar)
	Visit(visitor)
	if len(m) != 0 {
		t.Errorf("Visit sees unset envvars")
		for k, v := range m {
			t.Log(k, *v)
		}
	}
	// Now set all envvars
	Set("TEST_BOOL", "true")
	Set("TEST_INT", "1")
	Set("TEST_INT64", "1")
	Set("TEST_UINT", "1")
	Set("TEST_UINT64", "1")
	Set("TEST_STRING", "1")
	Set("TEST_FLOAT64", "1")
	Set("TEST_DURATION", "1s")
	desired = "1"
	Visit(visitor)
	if len(m) != 8 {
		t.Error("Visit fails after set")
		for k, v := range m {
			t.Log(k, *v)
		}
	}
	// Now test they're visited in sort order.
	var envVarNames []string
	Visit(func(ev *EnvVar) { envVarNames = append(envVarNames, ev.Name) })
	if !sort.StringsAreSorted(envVarNames) {
		t.Errorf("envvar names not sorted: %v", envVarNames)
	}
}

func TestGet(t *testing.T) {
	ResetForTesting()
	Bool("TEST_BOOL", true)
	Int("TEST_INT", 1)
	Int64("TEST_INT64", 2)
	Uint("TEST_UINT", 3)
	Uint64("TEST_UINT64", 4)
	String("TEST_STRING", "5")
	Float64("TEST_FLOAT64", 6)
	Duration("TEST_DURATION", 7)

	visitor := func(ev *EnvVar) {
		if len(ev.Name) > 5 && ev.Name[0:5] == "TEST_" {
			g, ok := ev.Value.(Getter)
			if !ok {
				t.Errorf("Visit: value does not satisfy Getter: %T", ev.Value)
				return
			}
			switch ev.Name {
			case "TEST_BOOL":
				ok = g.Get() == true
			case "TEST_INT":
				ok = g.Get() == int(1)
			case "TEST_INT64":
				ok = g.Get() == int64(2)
			case "TEST_UINT":
				ok = g.Get() == uint(3)
			case "TEST_UINT64":
				ok = g.Get() == uint64(4)
			case "TEST_STRING":
				ok = g.Get() == "5"
			case "TEST_FLOAT64":
				ok = g.Get() == float64(6)
			case "TEST_DURATION":
				ok = g.Get() == time.Duration(7)
			}
			if !ok {
				t.Errorf("Visit: bad value %T(%v) for %s", g.Get(), g.Get(), ev.Name)
			}
		}
	}
	VisitAll(visitor)
}

func testParse(ev *EnvVarSet, t *testing.T) {
	if ev.Parsed() {
		t.Error("ev.Parse() = true before Parse")
	}
	boolEnvVar := ev.Bool("BOOL", false)
	bool2EnvVar := ev.Bool("BOOL2", false)
	intEnvVar := ev.Int("INT", 0)
	int64EnvVar := ev.Int64("INT64", 0)
	uintEnvVar := ev.Uint("UINT", 0)
	uint64EnvVar := ev.Uint64("UINT64", 0)
	stringEnvVar := ev.String("STRING", "0")
	float64EnvVar := ev.Float64("FLOAT64", 0)
	durationEnvVar := ev.Duration("DURATION", 5*time.Second)
	args := []string{
		"BOOL=1",
		"BOOL2=true",
		"INT=22",
		"INT64=0x23",
		"UINT=24",
		"UINT64=25",
		"STRING=hello",
		"FLOAT64=2718e28",
		"DURATION=2m",
	}
	if err := ev.Parse(args); err != nil {
		t.Fatal(err)
	}
	if !ev.Parsed() {
		t.Error("ev.Parse() = false after Parse")
	}
	if *boolEnvVar != true {
		t.Error("bool envvar should be true, is ", *boolEnvVar)
	}
	if *bool2EnvVar != true {
		t.Error("bool2 envvar should be true, is ", *bool2EnvVar)
	}
	if *intEnvVar != 22 {
		t.Error("int envvar should be 22, is ", *intEnvVar)
	}
	if *int64EnvVar != 0x23 {
		t.Error("int64 envvar should be 0x23, is ", *int64EnvVar)
	}
	if *uintEnvVar != 24 {
		t.Error("uint envvar should be 24, is ", *uintEnvVar)
	}
	if *uint64EnvVar != 25 {
		t.Error("uint64 envvar should be 25, is ", *uint64EnvVar)
	}
	if *stringEnvVar != "hello" {
		t.Error("string envvar should be `hello`, is ", *stringEnvVar)
	}
	if *float64EnvVar != 2718e28 {
		t.Error("float64 envvar should be 2718e28, is ", *float64EnvVar)
	}
	if *durationEnvVar != 2*time.Minute {
		t.Error("duration envvar should be 2m, is ", *durationEnvVar)
	}
}

func TestEnvVarSetParse(t *testing.T) {
	testParse(NewEnvVarSet("test", ContinueOnError), t)
}

// Declare a user-defined envvar type.
type userVar []string

func (uv *userVar) String() string {
	return fmt.Sprint([]string(*uv))
}

func (uv *userVar) Set(value string) error {
	*uv = append(*uv, value)
	return nil
}

func TestUserDefined(t *testing.T) {
	var userVars EnvVarSet
	userVars.Init("test", ContinueOnError)
	var uv userVar
	userVars.Var(&uv, "UV")
	if err := userVars.Parse([]string{"UV=1", "UV=2", "UV=3", "NOTDEFINED=something"}); err != nil {
		t.Error(err)
	}
	if len(uv) != 3 {
		t.Fatal("expected 3 args; got ", len(uv))
	}
	expect := "[1 2 3]"
	if uv.String() != expect {
		t.Errorf("expected value %q got %q", expect, uv.String())
	}
}

// Declare a user-defined boolean envvar type.
type boolEnvVar struct {
	count int
}

func (b *boolEnvVar) String() string {
	return fmt.Sprintf("%d", b.count)
}

func (b *boolEnvVar) Set(value string) error {
	if v, _ := strconv.ParseBool(value); v == true {
		b.count++
	}
	return nil
}

func TestUserDefinedBool(t *testing.T) {
	var envVars EnvVarSet
	envVars.Init("test", ContinueOnError)
	var b boolEnvVar
	var err error
	envVars.Var(&b, "B")
	if err = envVars.Parse([]string{"B=1", "B=t", "B=T", "B=TRUE", "B=true", "B=True", "B=0", "B=false", "NOTDEFINED=something"}); err != nil {
		t.Error(err)
	}
	if b.count != 6 {
		t.Errorf("want: %d; got: %d", 6, b.count)
	}
}

// Issue 19230 (from original flag package https://github.com/golang/go/): validate range of
// int and Uint EnvVar values.
func TestIntEnvVarOverflow(t *testing.T) {
	if strconv.IntSize != 32 {
		return
	}
	ResetForTesting()
	Int("i", 0)
	Uint("u", 0)
	if err := Set("i", "2147483648"); err == nil {
		t.Error("unexpected success setting Int")
	}
	if err := Set("u", "4294967296"); err == nil {
		t.Error("unexpected success setting Uint")
	}
}
