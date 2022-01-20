package core

import (
	"strconv"
	"strings"
)

// Severity is the severity of the event described in a log entry. These
// guideline severity levels are ordered, with numerically smaller levels
// treated as less severe than numerically larger levels.
type Severity int

const (
	// Default means the log entry has no assigned severity level.
	Default Severity = iota
	// Debug means debug or trace information.
	Debug
	// Info means routine information, such as ongoing status or performance.
	Info
	// Notice means normal but significant events, such as start up, shut down, or configuration.
	Notice
	// Warning means events that might cause problems.
	Warning
	// Error means events that are likely to cause problems.
	Error
	// Critical means events that cause more severe problems or brief outages.
	Critical
	// Alert means a person must take an action immediately.
	Alert
	// Emergency means one or more systems are unusable.
	Emergency
)

var severityName = map[Severity]string{
	Default:   "default",
	Debug:     "debug",
	Info:      "info",
	Notice:    "notice",
	Warning:   "warning",
	Error:     "error",
	Critical:  "critical",
	Alert:     "alert",
	Emergency: "emergency",
}

// String converts a severity level to a string.
func (v Severity) String() string {
	// same as proto.EnumName
	s, ok := severityName[v]
	if ok {
		return s
	}
	return strconv.Itoa(int(v))
}

// Parse parse Severity whose name equals s, ignoring case. It
// sets Default if no Severity matches.
func (v *Severity) Parse(s string) {
	sl := strings.ToLower(s)
	for sev, name := range severityName {
		if strings.ToLower(name) == sl {
			*v = sev
			return
		}
	}
	*v = Default
}

// ParseSeverity returns the Severity whose name equals s, ignoring case. It
// returns Default if no Severity matches.
func ParseSeverity(s string) Severity {
	sl := strings.ToLower(s)
	for sev, name := range severityName {
		if strings.ToLower(name) == sl {
			return sev
		}
	}
	return Default
}

func SignToSeverity(v int) Severity {
	if v == 0 {
		return Notice
	} else if v < 0 {
		return Error
	}
	return Info
}
