package setup

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/stretchr/testify/require"
)

func TestPersonalPrivacy(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		for _, admin := range []AdminConfig{
			{},
			{Email: "admin@example.invalid"},
			{Password: "Strong-personal-password-123!"},
		} {
			before := admin
			emailGenerated, passwordGenerated, err := prepareAdminCredentials(&admin)
			require.ErrorContains(t, err, "ADMIN_EMAIL and ADMIN_PASSWORD must be set")
			require.False(t, emailGenerated)
			require.False(t, passwordGenerated)
			require.Equal(t, before, admin, "do not generate credentials that would be printed")
		}
		admin := AdminConfig{Email: "admin@example.invalid", Password: "Strong-personal-password-123!"}
		emailGenerated, passwordGenerated, err := prepareAdminCredentials(&admin)
		require.NoError(t, err)
		require.False(t, emailGenerated)
		require.False(t, passwordGenerated)
		admin.Password = "weak"
		_, _, err = prepareAdminCredentials(&admin)
		require.Error(t, err, "retain upstream password validation")
	})
}
