package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/chromedp/chromedp"
)

func evalResultToString(ctx context.Context, script string) (string, error) {
	var raw []byte
	err := chromedp.Run(ctx, chromedp.Evaluate(script, &raw))
	if err != nil {
		return "", err
	}
	var v interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &v)
	}
	return extractString(v), nil
}

func extractString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if m, ok := v.(map[string]interface{}); ok {
		if res, ok := m["result"].(map[string]interface{}); ok {
			if val, ok := res["value"].(string); ok {
				return val
			}
			if val, ok := res["Value"].(string); ok {
				return val
			}
			if val := res["value"]; val != nil && fmt.Sprint(val) != "map[]" {
				return fmt.Sprint(val)
			}
		}
		if val, ok := m["value"].(string); ok {
			return val
		}
		if val, ok := m["Value"].(string); ok {
			return val
		}
		if val := m["value"]; val != nil {
			s := fmt.Sprint(val)
			if s != "" && s != "map[]" {
				return s
			}
		}
		if val := m["Value"]; val != nil {
			s := fmt.Sprint(val)
			if s != "" && s != "map[]" {
				return s
			}
		}
		return ""
	}
	return fmt.Sprint(v)
}

func evalResultToBool(ctx context.Context, script string) (bool, error) {
	var raw []byte
	err := chromedp.Run(ctx, chromedp.Evaluate(script, &raw))
	if err != nil {
		return false, err
	}
	var v interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &v)
	}
	if b, ok := v.(bool); ok {
		return b, nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		if val, ok := m["value"].(bool); ok {
			return val, nil
		}
	}
	return false, nil
}

func chromedpLogf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(msg, "could not unmarshal event:") {
		return
	}
	log.Print(msg)
}
