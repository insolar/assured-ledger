package throw

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeverityOf(t *testing.T) {
	require.Equal(t, NormalSeverity, SeverityOf(New("test")))
	require.Equal(t, NormalSeverity, SeverityOf(Severe(NormalSeverity, "test")))
	require.Equal(t, BlameSeverity, SeverityOf(Blame("test")))
	require.Equal(t, ViolationSeverity, SeverityOf(Violation("test")))
	require.Equal(t, FraudSeverity, SeverityOf(Fraud("test")))
	require.Equal(t, RemoteBreachSeverity, SeverityOf(RemoteBreach("test")))
	require.Equal(t, LocalBreachSeverity, SeverityOf(LocalBreach("test")))
	require.Equal(t, LocalBreachSeverity, SeverityOf(DeadCanary("test")))
	require.Equal(t, FatalSeverity, SeverityOf(Fatal("test")))
}

func TestWithSeverity(t *testing.T) {
	require.Equal(t, NormalSeverity, SeverityOf(New("test")))
	require.Equal(t, FraudSeverity, SeverityOf(WithSeverity(Blame("test"), FraudSeverity)))
	require.Equal(t, NormalSeverity, SeverityOf(WithSeverity(Blame("test"), NormalSeverity)))
}

func TestWithDefaultSeverity(t *testing.T) {
	require.Equal(t, BlameSeverity, SeverityOf(WithDefaultSeverity(Blame("test"), FraudSeverity)))
	require.Equal(t, BlameSeverity, SeverityOf(WithDefaultSeverity(Blame("test"), NormalSeverity)))
}

func TestWithEscalatedSeverity(t *testing.T) {
	require.Equal(t, FraudSeverity, SeverityOf(WithEscalatedSeverity(Blame("test"), FraudSeverity)))
	require.Equal(t, BlameSeverity, SeverityOf(WithEscalatedSeverity(Blame("test"), NormalSeverity)))
}

func TestGetSeverity(t *testing.T) {
	s, ok := GetSeverity(New("test"))
	require.False(t, ok)
	require.Equal(t, NormalSeverity, s)

	s, ok = GetSeverity(Blame("test"))
	require.True(t, ok)
	require.Equal(t, BlameSeverity, s)
}
