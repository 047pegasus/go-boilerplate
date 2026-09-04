Hello this is my personal golang backend focused boilerplate. Heavily inspired from the implementation from Sriniously (https://youtu.be/E4CSP_KixPM)

Steps to replicate before running the project:

a.) Make sure Taskfile is installed from https://taskfile.dev/docs/installation.
b.) We will be using the following 3rd party/vendor packages in our boierplate:

    1.) Koanf (for Config Management): https://github.com/knadh/koanf
    2.) Zerolog (for logging purposes): https://github.com/rs/zerolog
    3.) Validator (for all validations needs): https://github.com/go-playground/validator
    4.) GoDotEnv (for autolading env files): https://github.com/joho/godotenv
    5.) Sentry-Go (for observability needs): https://github.com/getsentry/sentry-go
    6.) Tern (for managing DB migrations): https://github.com/jackc/tern
    7.) Turborepo (for managing entire monorepo setup): https://turborepo.dev/
    8.) Asynq (for managing async background jobs using Redis): https://github.com/hibiken/asynq
    9.) Resend (for sending emails): https://resend.com/

Instead of using NewRelic since we are using Sentry in my project we need to have something to make a drop in replacement for [github.com/newrelic/go-agent/v3/integrations/nrpgx5][nrpgx]
So I made a drop in replacement implementation in a file called [sentry_tracer.go] in the database package.

I did not go ahead with the sentry-go SQL instrumentation (https://docs.sentry.io/platforms/go/tracing/instrumentation/sql/) writer implementation
as they suggested to use it in the Sentry Docs to keep the code format intact with the original newRelic implementation (so that newRelic can be used inplace quickly)
and instead I made my custom writer using Zerolog.