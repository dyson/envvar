// Copyright 2017 Dyson Simmons. All rights reserved.

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package envvar implements environment variable parsing.

Usage:

Define environment variables using envvar.String(), Bool(), Int(), etc.

This declares an integer envvar, ENVVARNAME, stored in the pointer ip, with type *int.
	import "envvar"
	var ip = envvar.Int("ENVVARNAME", 1234)
If you like, you can bind the envvar to a variable using the Var() functions.
	var i int
	func init() {
		envvar.IntVar(&i, "ENVVARNAME", 1234)
	}
Or you can create custom envvars that satisfy the Value interface (with
pointer receivers) and couple them to environment variable parsing by
	envvar.Var(&envVarVal, "ENVVARNAME")
For such envvars, the default value is just the initial value of the variable.

After all envvars are defined, call
	envvar.Parse()
to parse the environment variables into the defined envvars.

Envvars may then be used directly. If you're using the envvars themselves,
they are all pointers; if you bind to variables, they're values.
	fmt.Println("ip has value ", *ip)
	fmt.Println("i has value ", i)

Integer envvars accept 1234, 0664, 0x1234 and may be negative.
Boolean envvars may be:
	1, 0, t, f, T, F, true, false, TRUE, FALSE, True, False
Duration envvars accept any input valid for time.ParseDuration.

The default set of envvars is controlled by top-level functions.
The EnvVarSet type allows one to define	independent sets of envvars,
which facilitates their independent parsing. The methods of EnvVarSet
are	analogous to the top-level functions for the default	envvar set.
*/
package envvar

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"
)

// -- bool Value
type boolValue bool

func newBoolValue(val bool, p *bool) *boolValue {
	*p = val
	return (*boolValue)(p)
}

func (b *boolValue) Set(s string) error {
	v, err := strconv.ParseBool(s)
	*b = boolValue(v)
	return err
}

func (b *boolValue) Get() interface{} { return bool(*b) }

func (b *boolValue) String() string { return strconv.FormatBool(bool(*b)) }

// -- int Value
type intValue int

func newIntValue(val int, p *int) *intValue {
	*p = val
	return (*intValue)(p)
}

func (i *intValue) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, strconv.IntSize)
	*i = intValue(v)
	return err
}

func (i *intValue) Get() interface{} { return int(*i) }

func (i *intValue) String() string { return strconv.Itoa(int(*i)) }

// -- int64 Value
type int64Value int64

func newInt64Value(val int64, p *int64) *int64Value {
	*p = val
	return (*int64Value)(p)
}

func (i *int64Value) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 64)
	*i = int64Value(v)
	return err
}

func (i *int64Value) Get() interface{} { return int64(*i) }

func (i *int64Value) String() string { return strconv.FormatInt(int64(*i), 10) }

// -- uint Value
type uintValue uint

func newUintValue(val uint, p *uint) *uintValue {
	*p = val
	return (*uintValue)(p)
}

func (i *uintValue) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, strconv.IntSize)
	*i = uintValue(v)
	return err
}

func (i *uintValue) Get() interface{} { return uint(*i) }

func (i *uintValue) String() string { return strconv.FormatUint(uint64(*i), 10) }

// -- uint64 Value
type uint64Value uint64

func newUint64Value(val uint64, p *uint64) *uint64Value {
	*p = val
	return (*uint64Value)(p)
}

func (i *uint64Value) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 64)
	*i = uint64Value(v)
	return err
}

func (i *uint64Value) Get() interface{} { return uint64(*i) }

func (i *uint64Value) String() string { return strconv.FormatUint(uint64(*i), 10) }

// -- string Value
type stringValue string

func newStringValue(val string, p *string) *stringValue {
	*p = val
	return (*stringValue)(p)
}

func (s *stringValue) Set(val string) error {
	*s = stringValue(val)
	return nil
}

func (s *stringValue) Get() interface{} { return string(*s) }

func (s *stringValue) String() string { return string(*s) }

// -- float64 Value
type float64Value float64

func newFloat64Value(val float64, p *float64) *float64Value {
	*p = val
	return (*float64Value)(p)
}

func (f *float64Value) Set(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	*f = float64Value(v)
	return err
}

func (f *float64Value) Get() interface{} { return float64(*f) }

func (f *float64Value) String() string { return strconv.FormatFloat(float64(*f), 'g', -1, 64) }

// -- time.Duration Value
type durationValue time.Duration

func newDurationValue(val time.Duration, p *time.Duration) *durationValue {
	*p = val
	return (*durationValue)(p)
}

func (d *durationValue) Set(s string) error {
	v, err := time.ParseDuration(s)
	*d = durationValue(v)
	return err
}

func (d *durationValue) Get() interface{} { return time.Duration(*d) }

func (d *durationValue) String() string { return (*time.Duration)(d).String() }

// Value is the interface to the dynamic value stored in a EnvVar.
// (The default value is represented as a string.)
//
// Set is called once for each EnvVar present.
// The envvar package may call the String method with a zero-valued receiver,
// such as a nil pointer.
type Value interface {
	String() string
	Set(string) error
}

// Getter is an interface that allows the contents of a Value to be retrieved.
// It wraps the Value interface, rather than being part of it, because it
// appeared after Go 1 and its compatibility rules. All Value types provided
// by this package satisfy the Getter interface.
type Getter interface {
	Value
	Get() interface{}
}

// ErrorHandling defines how EnvVarSet.Parse behaves if the parse fails.
type ErrorHandling int

// These constants cause EnvVarSet.Parse to behave as described if the parse fails.
const (
	ContinueOnError ErrorHandling = iota // return a descriptive error.
	ExitOnError                          // call os.Exit(2).
	PanicOnError                         // call panic with a descriptive error.
)

// A EnvVarSet represents a set of defined envVars. The zero value of a EnvVarSet
// has no name and has ContinueOnError error handling.
type EnvVarSet struct {
	name          string
	parsed        bool
	actual        map[string]*EnvVar
	formal        map[string]*EnvVar
	errorHandling ErrorHandling
	output        io.Writer // nil means stderr; use out() accessor
}

// A EnvVar represents the state of a EnvVar.
type EnvVar struct {
	Name  string // name of environment variable
	Value Value  // value as set
}

// sortEnvVars returns the EnvVars as a slice in lexicographical sorted order.
func sortEnvVars(envVars map[string]*EnvVar) []*EnvVar {
	list := make(sort.StringSlice, len(envVars))
	i := 0
	for _, ev := range envVars {
		list[i] = ev.Name
		i++
	}
	list.Sort()
	result := make([]*EnvVar, len(list))
	for i, name := range list {
		result[i] = envVars[name]
	}
	return result
}

func (evs *EnvVarSet) out() io.Writer {
	if evs.output == nil {
		return os.Stderr
	}
	return evs.output
}

// SetOutput sets the destination for error messages.
// If output is nil, os.Stderr is used.
func (evs *EnvVarSet) SetOutput(output io.Writer) {
	evs.output = output
}

// VisitAll visits the sets EnvVars in lexicographical order, calling
// fn for each. It visits all EnvVars, even those not set.
func (evs *EnvVarSet) VisitAll(fn func(*EnvVar)) {
	for _, envVar := range sortEnvVars(evs.formal) {
		fn(envVar)
	}
}

// VisitAll visits the default sets EnvVars in lexicographical order,
// calling fn for each. It visits EnvVars, even those not set.
func VisitAll(fn func(*EnvVar)) {
	EnvVars.VisitAll(fn)
}

// Visit visits the sets EnvVars in lexicographical order, calling fn for each.
// It visits only those EnvVars that have been set.
func (evs *EnvVarSet) Visit(fn func(*EnvVar)) {
	for _, envVar := range sortEnvVars(evs.actual) {
		fn(envVar)
	}
}

// Visit visits the default sets EnvVars in lexicographical order,
// calling fn for each. It visits only those EnvVars that have been set.
func Visit(fn func(*EnvVar)) {
	EnvVars.Visit(fn)
}

// Lookup returns the EnvVar structure of the named EnvVar,
// returning nil if none exists.
func (evs *EnvVarSet) Lookup(name string) *EnvVar {
	return evs.formal[name]
}

// Lookup returns the EnvVar structure of the named EnvVar,
// returning nil if none exists.
func Lookup(name string) *EnvVar {
	return EnvVars.formal[name]
}

// Set sets the value of the named EnvVar.
func (evs *EnvVarSet) Set(name, value string) error {
	envVar, ok := evs.formal[name]
	if !ok {
		return fmt.Errorf("no such environment variable %v", name)
	}
	err := envVar.Value.Set(value)
	if err != nil {
		return err
	}
	if evs.actual == nil {
		evs.actual = make(map[string]*EnvVar)
	}
	evs.actual[name] = envVar
	return nil
}

// Set sets the value of the named EnvVar for the default set.
func Set(name, value string) error {
	return EnvVars.Set(name, value)
}

// NEnvVar returns the number of EnvVars that have been defined.
func (evs *EnvVarSet) NEnvVar() int { return len(evs.actual) }

// NEnvVar returns the number of EnvVars that have been defined.
func NEnvVar() int { return len(EnvVars.actual) }

// BoolVar defines a bool EnvVar with specified name, and default value.
// The argument p points to a bool variable in which to store the value of the EnvVar.
func (evs *EnvVarSet) BoolVar(p *bool, name string, value bool) {
	evs.Var(newBoolValue(value, p), name)
}

// BoolVar defines a bool EnvVar with specified name, and default value.
// The argument p points to a bool variable in which to store the value of the EnvVar.
func BoolVar(p *bool, name string, value bool) {
	EnvVars.Var(newBoolValue(value, p), name)
}

// Bool defines a bool EnvVar with specified name, and default value.
// The return value is the address of a bool variable that stores the value of the EnvVar.
func (evs *EnvVarSet) Bool(name string, value bool) *bool {
	p := new(bool)
	evs.BoolVar(p, name, value)
	return p
}

// Bool defines a bool EnvVar with specified name, and default value.
// The return value is the address of a bool variable that stores the value of the EnvVar.
func Bool(name string, value bool) *bool {
	return EnvVars.Bool(name, value)
}

// IntVar defines an int EnvVar with specified name, and default value.
// The argument p points to an int variable in which to store the value of the EnvVar.
func (evs *EnvVarSet) IntVar(p *int, name string, value int) {
	evs.Var(newIntValue(value, p), name)
}

// IntVar defines an int EnvVar with specified name, and default value.
// The argument p points to an int variable in which to store the value of the EnvVar.
func IntVar(p *int, name string, value int) {
	EnvVars.Var(newIntValue(value, p), name)
}

// Int defines an int EnvVar with specified name, and default value.
// The return value is the address of an int variable that stores the value of the EnvVar.
func (evs *EnvVarSet) Int(name string, value int) *int {
	p := new(int)
	evs.IntVar(p, name, value)
	return p
}

// Int defines an int EnvVar with specified name, and default value.
// The return value is the address of an int variable that stores the value of the EnvVar.
func Int(name string, value int) *int {
	return EnvVars.Int(name, value)
}

// Int64Var defines an int64 EnvVar with specified name, and default value.
// The argument p points to an int64 variable in which to store the value of the EnvVar.
func (evs *EnvVarSet) Int64Var(p *int64, name string, value int64) {
	evs.Var(newInt64Value(value, p), name)
}

// Int64Var defines an int64 EnvVar with specified name, and default value.
// The argument p points to an int64 variable in which to store the value of the EnvVar.
func Int64Var(p *int64, name string, value int64) {
	EnvVars.Var(newInt64Value(value, p), name)
}

// Int64 defines an int64 EnvVar with specified name, and default value.
// The return value is the address of an int64 variable that stores the value of the EnvVar.
func (evs *EnvVarSet) Int64(name string, value int64) *int64 {
	p := new(int64)
	evs.Int64Var(p, name, value)
	return p
}

// Int64 defines an int64 EnvVar with specified name, and default value.
// The return value is the address of an int64 variable that stores the value of the EnvVar.
func Int64(name string, value int64) *int64 {
	return EnvVars.Int64(name, value)
}

// UintVar defines a uint EnvVar with specified name, and default value.
// The argument p points to a uint variable in which to store the value of the EnvVar.
func (evs *EnvVarSet) UintVar(p *uint, name string, value uint) {
	evs.Var(newUintValue(value, p), name)
}

// UintVar defines a uint EnvVar with specified name, and default value.
// The argument p points to a uint  variable in which to store the value of the EnvVar.
func UintVar(p *uint, name string, value uint) {
	EnvVars.Var(newUintValue(value, p), name)
}

// Uint defines a uint EnvVar with specified name, and default value.
// The return value is the address of a uint  variable that stores the value of the EnvVar.
func (evs *EnvVarSet) Uint(name string, value uint) *uint {
	p := new(uint)
	evs.UintVar(p, name, value)
	return p
}

// Uint defines a uint EnvVar with specified name, and default value.
// The return value is the address of a uint  variable that stores the value of the EnvVar.
func Uint(name string, value uint) *uint {
	return EnvVars.Uint(name, value)
}

// Uint64Var defines a uint64 EnvVar with specified name, and default value.
// The argument p points to a uint64 variable in which to store the value of the EnvVar.
func (evs *EnvVarSet) Uint64Var(p *uint64, name string, value uint64) {
	evs.Var(newUint64Value(value, p), name)
}

// Uint64Var defines a uint64 EnvVar with specified name, and default value.
// The argument p points to a uint64 variable in which to store the value of the EnvVar.
func Uint64Var(p *uint64, name string, value uint64) {
	EnvVars.Var(newUint64Value(value, p), name)
}

// Uint64 defines a uint64 EnvVar with specified name, and default value.
// The return value is the address of a uint64 variable that stores the value of the EnvVar.
func (evs *EnvVarSet) Uint64(name string, value uint64) *uint64 {
	p := new(uint64)
	evs.Uint64Var(p, name, value)
	return p
}

// Uint64 defines a uint64 EnvVar with specified name, and default value.
// The return value is the address of a uint64 variable that stores the value of the EnvVar.
func Uint64(name string, value uint64) *uint64 {
	return EnvVars.Uint64(name, value)
}

// StringVar defines a string EnvVar with specified name, and default value.
// The argument p points to a string variable in which to store the value of the EnvVar.
func (evs *EnvVarSet) StringVar(p *string, name string, value string) {
	evs.Var(newStringValue(value, p), name)
}

// StringVar defines a string EnvVar with specified name, and default value.
// The argument p points to a string variable in which to store the value of the EnvVar.
func StringVar(p *string, name string, value string) {
	EnvVars.Var(newStringValue(value, p), name)
}

// String defines a string EnvVar with specified name, and default value.
// The return value is the address of a string variable that stores the value of the EnvVar.
func (evs *EnvVarSet) String(name string, value string) *string {
	p := new(string)
	evs.StringVar(p, name, value)
	return p
}

// String defines a string EnvVar with specified name, and default value.
// The return value is the address of a string variable that stores the value of the EnvVar.
func String(name string, value string) *string {
	return EnvVars.String(name, value)
}

// Float64Var defines a float64 EnvVar with specified name, and default value.
// The argument p points to a float64 variable in which to store the value of the EnvVar.
func (evs *EnvVarSet) Float64Var(p *float64, name string, value float64) {
	evs.Var(newFloat64Value(value, p), name)
}

// Float64Var defines a float64 EnvVar with specified name, and default value.
// The argument p points to a float64 variable in which to store the value of the EnvVar.
func Float64Var(p *float64, name string, value float64) {
	EnvVars.Var(newFloat64Value(value, p), name)
}

// Float64 defines a float64 EnvVar with specified name, and default value.
// The return value is the address of a float64 variable that stores the value of the EnvVar.
func (evs *EnvVarSet) Float64(name string, value float64) *float64 {
	p := new(float64)
	evs.Float64Var(p, name, value)
	return p
}

// Float64 defines a float64 EnvVar with specified name, and default value.
// The return value is the address of a float64 variable that stores the value of the EnvVar.
func Float64(name string, value float64) *float64 {
	return EnvVars.Float64(name, value)
}

// DurationVar defines a time.Duration EnvVar with specified name, and default value.
// The argument p points to a time.Duration variable in which to store the value of the EnvVar.
// The EnvVar accepts a value acceptable to time.ParseDuration.
func (evs *EnvVarSet) DurationVar(p *time.Duration, name string, value time.Duration) {
	evs.Var(newDurationValue(value, p), name)
}

// DurationVar defines a time.Duration EnvVar with specified name, and default value.
// The argument p points to a time.Duration variable in which to store the value of the EnvVar.
// The EnvVar accepts a value acceptable to time.ParseDuration.
func DurationVar(p *time.Duration, name string, value time.Duration) {
	EnvVars.Var(newDurationValue(value, p), name)
}

// Duration defines a time.Duration EnvVar with specified name, and default value.
// The return value is the address of a time.Duration variable that stores the value of the EnvVar.
// The EnvVar accepts a value acceptable to time.ParseDuration.
func (evs *EnvVarSet) Duration(name string, value time.Duration) *time.Duration {
	p := new(time.Duration)
	evs.DurationVar(p, name, value)
	return p
}

// Duration defines a time.Duration EnvVar with specified name, and default value.
// The return value is the address of a time.Duration variable that stores the value of the EnvVar.
// The EnvVar accepts a value acceptable to time.ParseDuration.
func Duration(name string, value time.Duration) *time.Duration {
	return EnvVars.Duration(name, value)
}

// Var defines a EnvVar with the specified name. The type and value of the EnvVar
// are represented by the first argument, of type Value, which typically holds a
// user-defined implementation of Value. For instance, the caller could create a
// EnvVar that turns a comma-separated string into a slice of strings by giving
// the slice the methods of Value; in particular, Set would decompose the
// comma-separated string into the slice.
func (evs *EnvVarSet) Var(value Value, name string) {
	envVar := &EnvVar{name, value}
	_, alreadythere := evs.formal[name]
	if alreadythere {
		var msg string
		if evs.name == "" {
			msg = fmt.Sprintf("EnvVar redefined: %s", name)
		} else {
			msg = fmt.Sprintf("%s sets EnvVar redefined: %s", evs.name, name)
		}
		fmt.Fprintln(evs.out(), msg)
		panic(msg) // happens only if env vars are declared with identical names
	}
	if evs.formal == nil {
		evs.formal = make(map[string]*EnvVar)
	}
	evs.formal[name] = envVar
}

// Var defines a env var with the specified name. The type and
// value of the env var are represented by the first argument, of type Value, which
// typically holds a user-defined implementation of Value. For instance, the
// caller could create a env var that turns a comma-separated string into a slice
// of strings by giving the slice the methods of Value; in particular, Set would
// decompose the comma-separated string into the slice.
func Var(value Value, name string) {
	EnvVars.Var(value, name)
}

// failf prints to standard error a formatted error and returns the error.
func (evs *EnvVarSet) failf(format string, a ...interface{}) error {
	err := fmt.Errorf(format, a...)
	fmt.Fprintln(evs.out(), err)
	return err
}

// parseOne parses one env var. It reports whether a env var was seen.
func (evs *EnvVarSet) parseOne(envString string) error {
	name := ""
	value := ""
	for i := 1; i < len(envString); i++ { // equals cannot be first
		if envString[i] == '=' {
			value = envString[i+1:]
			name = envString[0:i]
			break
		}
	}
	envVar, alreadythere := evs.formal[name]
	if !alreadythere { // skip this env var as we haven't defined it in the set
		return nil
	}
	if err := envVar.Value.Set(value); err != nil {
		return evs.failf("invalid value %q for env var %s: %v", value, name, err)
	}
	if evs.actual == nil {
		evs.actual = make(map[string]*EnvVar)
	}
	evs.actual[name] = envVar
	return nil
}

// Parse parses all env var definitions. Must be called after all env vars in
// the EnvVarSet are defined and before env vars are accessed by the program.
func (evs *EnvVarSet) Parse(environment []string) error {
	evs.parsed = true
	for _, envString := range environment {
		err := evs.parseOne(envString)
		if err != nil {
			switch evs.errorHandling {
			case ContinueOnError:
				return err
			case ExitOnError:
				os.Exit(2)
			case PanicOnError:
				panic(err)
			}
		}
	}
	return nil
}

// Parsed reports whether evs.Parse has been called.
func (evs *EnvVarSet) Parsed() bool {
	return evs.parsed
}

// Parse parses the env vars from os.Environ().  Must be called
// after all env vars are defined and before env vars are accessed by the program.
func Parse() {
	EnvVars.Parse(os.Environ())
}

// Parsed reports whether the env vars have been parsed.
func Parsed() bool {
	return EnvVars.Parsed()
}

// EnvVars is the default set of env vars, parsed from os.Environ().
// The top-level functions such as BoolVar, Arg, and so on are wrappers for the
// methods of EnvVars.
var EnvVars = NewEnvVarSet(os.Args[0], ExitOnError)

// NewEnvVarSet returns a new, empty env var set with the specified name and
// error handling property.
func NewEnvVarSet(name string, errorHandling ErrorHandling) *EnvVarSet {
	evs := &EnvVarSet{
		name:          name,
		errorHandling: errorHandling,
	}
	return evs
}

// Init sets the name and error handling property for a env var set.
// By default, the zero EnvVarSet uses an empty name and the
// ContinueOnError error handling policy.
func (evs *EnvVarSet) Init(name string, errorHandling ErrorHandling) {
	evs.name = name
	evs.errorHandling = errorHandling
}
