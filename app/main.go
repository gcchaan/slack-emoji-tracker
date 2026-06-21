package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/slack-go/slack"
)

func handler(ctx context.Context, event events.EventBridgeEvent) error {
	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		err := errors.New("S3_BUCKET_NAME environment variable is required")
		slog.Error(err.Error())
		return err
	}
	objectKey := os.Getenv("S3_OBJECT_KEY")
	if objectKey == "" {
		err := errors.New("S3_OBJECT_KEY environment variable is required")
		slog.Error(err.Error())
		return err
	}
	channelID := os.Getenv("SLACK_CHANNEL_ID")
	if channelID == "" {
		err := errors.New("SLACK_CHANNEL_ID environment variable is required")
		slog.Error(err.Error())
		return err
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		slog.Error("Failed to load AWS config", "error", err)
		return err
	}

	ssmClient := ssm.NewFromConfig(cfg)
	token, err := fetchSlackToken(ctx, ssmClient)
	if err != nil {
		slog.Error("Failed to fetch Slack token", "error", err)
		return err
	}
	api := slack.New(token)

	s3Client := s3.NewFromConfig(cfg)
	cachedEmojis, found, err := LoadCacheFromS3(ctx, s3Client, bucketName, objectKey)
	if err != nil {
		slog.Error("Failed to retrieve cached emojis", "error", err)
		return err
	}

	currentEmojis, err := listSlackEmojis(api)
	if err != nil {
		slog.Error("Failed to list Slack emojis", "error", err)
		return err
	}

	if !found {
		slog.Info("No cached emojis found; treating as first-time run")
		if err := SaveCacheToS3(ctx, s3Client, bucketName, objectKey, currentEmojis); err != nil {
			slog.Error("Failed to save initial emoji cache", "error", err)
			return err
		}
		return nil
	}

	added, deleted := diffSortedSlices(cachedEmojis, currentEmojis)
	if len(added) == 0 && len(deleted) == 0 {
		slog.Info("No changes in emojis; skipping Slack notification")
		return nil
	}
	message := buildSlackMessage(added, deleted)

	if err := SaveCacheToS3(ctx, s3Client, bucketName, objectKey, currentEmojis); err != nil {
		slog.Error("Failed to update emoji cache in S3", "error", err)
		return err
	}

	if err := postToSlack(api, channelID, message); err != nil {
		slog.Error("Failed to post update message to Slack", "error", err)
		return err
	}

	slog.Info("Emoji cache updated and Slack notification sent successfully")
	return nil
}

func fetchSlackToken(ctx context.Context, client *ssm.Client) (string, error) {
	paramName := os.Getenv("SLACK_TOKEN_SSM_PATH")
	if paramName == "" {
		return "", errors.New("SLACK_TOKEN_SSM_PATH environment variable is required")
	}

	out, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &paramName,
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("get parameter from SSM failed: %w", err)
	}
	return *out.Parameter.Value, nil
}

func SaveCacheToS3(ctx context.Context, client *s3.Client, bucket, key string, data []string) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal cache to JSON failed: %w", err)
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(jsonData),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("put object to S3 failed: %w", err)
	}

	return nil
}

func LoadCacheFromS3(ctx context.Context, client *s3.Client, bucket, key string) ([]string, bool, error) {
	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if _, ok := errors.AsType[*types.NoSuchKey](err); ok {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("get object from S3 failed: %w", err)
	}
	defer output.Body.Close()

	var data []string
	decoder := json.NewDecoder(output.Body)
	if err := decoder.Decode(&data); err != nil {
		return nil, false, fmt.Errorf("unmarshal cache from JSON failed: %w", err)
	}

	return data, true, nil
}

func listSlackEmojis(api *slack.Client) ([]string, error) {
	emojis, err := api.GetEmoji()
	if err != nil {
		return nil, fmt.Errorf("get emoji from Slack failed: %w", err)
	}

	keys := slices.Collect(maps.Keys(emojis))
	slices.Sort(keys)

	return keys, nil
}

func formatEmojis(emojis []string) string {
	formatted := make([]string, len(emojis))
	for i, e := range emojis {
		formatted[i] = ":" + e + ":"
	}
	return strings.Join(formatted, " ")
}

func buildSlackMessage(added, removed []string) string {
	var message strings.Builder
	message.WriteString("Slack Emoji Tracker Update:\n")

	if len(added) > 0 {
		message.WriteString("Added Emojis:\n")
		message.WriteString(formatEmojis(added))
		message.WriteString("\n")
	} else {
		message.WriteString("No new emojis added.\n")
	}

	if len(removed) > 0 {
		message.WriteString("Removed Emojis:\n")
		message.WriteString(formatEmojis(removed))
		message.WriteString("\n")
	} else {
		message.WriteString("No emojis removed.\n")
	}

	return message.String()
}

func postToSlack(api *slack.Client, channelID string, message string) error {
	_, _, err := api.PostMessage(
		channelID,
		slack.MsgOptionText(message, false),
	)
	if err != nil {
		return fmt.Errorf("post message failed: %w", err)
	}

	return nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	lambda.Start(handler)
}
