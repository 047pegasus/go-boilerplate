Hello this is my personal golang backend focused boilerplate. Heavily inspired from the implementation from Sriniously (https://youtu.be/E4CSP_KixPM)

Steps to replicate before running the project:

a.) Make sure Taskfile is installed from https://taskfile.dev/docs/installation.
b.) We will be using the following 3rd party/vendor packages in our boierplate:

    1.) Koanf (for COnfig Management): https://github.com/knadh/koanf
    2.) Zerolog (for logging purposes): https://github.com/rs/zerolog
    3.) Validator (for all validations needs): https://github.com/go-playground/validator
    4.) GoDotEnv (for autolading env files): https://github.com/joho/godotenv
    5.) Sentry-Go (for observability needs): https://github.com/getsentry/sentry-go