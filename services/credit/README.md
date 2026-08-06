# credit (linux-do) points settlement — phase 2
# Reference: https://github.com/linux-do/credit
#
# Integration notes (confirmed from source):
# - credit is an OAuth2/OIDC client; the IdP must provide numeric uint64 user IDs →
#   use the Casdoor numeric-ID config
# - Deployment: PostgreSQL 18+ / Redis 6+ / Go 1.26; api+scheduler+worker processes + Next.js frontend
# - Cross-platform points: merchant model (API Key + signature) distribution/orders/transfers;
#   the forum joins as a merchant
# This phase only reserves the directory; no deployment yet.
