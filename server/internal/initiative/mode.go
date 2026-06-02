package initiative

// Mode is an Initiative's planning-time autonomy choice (ADR-0005). AFK is one
// big autonomous PR reviewed once; HITL is several smaller PRs reviewed in
// sequence. Mode records the choice; the runtime safety net is the tripwire.
type Mode string

const (
	ModeHITL Mode = "hitl"
	ModeAFK  Mode = "afk"
)

// ValidMode reports whether m is a known Mode.
func ValidMode(m Mode) bool {
	return m == ModeHITL || m == ModeAFK
}
