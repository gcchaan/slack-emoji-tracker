# slack-emoji-tracker

## Overview

This is a serverless application built with **Go** and **AWS SAM** that automatically tracks changes to your Slack workspace's custom emojis. 

### How it works:
1. **Trigger:** An EventBridge Scheduler triggers the Lambda function hourly during daytime.
2. **Fetch & Compare:** It fetches the current emoji list via the Slack API and compares it with the previous state cached in **Amazon S3**.
3. **Notify:** If any emojis were added or removed, it sends a nicely formatted summary message to your designated Slack channel.

## Requirements

* AWS CLI already configured with Administrator permission
* [Golang](https://golang.org)
* SAM CLI - [Install the SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)

## Setup process

### Create a new Slack app

`./assets/manifest.json` file helps you to create a new app with the required permissions.

You need to invite the Bot app to the channel where you want to post messages in advance (/invite @AppName).

### Create a new SSM parameter

You can get your token from [Slack API](https://api.slack.com/apps) page after creating a new app and adding the required permissions.

```bash
aws ssm put-parameter \
    --name "/slack-emoji-tracker/slack-bot-user-oauth-token" \
    --value "xoxb-***********-**************-************************" \
    --type "SecureString"
```

### Set up environment variables

samconfig.toml file is used to set up environment variables for your Lambda function. You can update the `parameter_overrides` section with your SSM parameter name.

```toml
[default.deploy.parameters]
region = "us-east-1"
parameter_overrides = [
  "SlackChannelId=C0123456789",
]
```

### Build and deploy

```bash
sam build
```

```bash
sam deploy --guided
```

### Testing

We use `testing` package that is built-in in Golang and you can simply run the following command to run our tests:

```shell
cd ./app/
go test -v .
```
