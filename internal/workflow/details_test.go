package workflow

import (
	"testing"
)

func TestParseDetails(t *testing.T) {
	details, err := ParseDetails([]byte(`
on:
  push:
  pull_request:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ${{ secrets.A }} ${{ env.APP_ENV }} ${{ vars.REGION }}
  test:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - run: echo ${{ secrets.B }} ${{ env.RUNTIME }} ${{ vars.APP_NAME }}
`))
	if err != nil {
		t.Fatalf("ParseDetails() error: %v", err)
	}
	if len(details.Events) != 2 || len(details.Jobs) != 2 || len(details.Secrets) != 2 || len(details.Env) != 2 || len(details.Vars) != 2 {
		t.Fatalf("unexpected parsed details: %+v", details)
	}
}
