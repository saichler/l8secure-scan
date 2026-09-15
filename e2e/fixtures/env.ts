// Environment configuration for the e2e suite. Every value has a default
// matching this project's own KIND cluster / seeded mock data (see
// go/tests/mocks/seed.go and go/run-local.sh), overridable via env vars so
// the suite can also point at a different deployment without code changes.

export const BASE_URL = process.env.BASE_URL || 'https://172.18.0.6:2790';

export const OPSADMIN_USER = process.env.OPSADMIN_USER || 'opsadmin';
export const OPSADMIN_PASS = process.env.OPSADMIN_PASS || 'opsadmin';

// The already-provisioned customer-role user for tenant "local"
// (go/tests/mocks/seed.go's localUserPassword constant).
export const CUSTOMER_USER = process.env.CUSTOMER_USER || 'local-user';
export const CUSTOMER_PASS = process.env.CUSTOMER_PASS || 'Vx9!TangoQm';
