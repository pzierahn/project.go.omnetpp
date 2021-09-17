package omnetpp

import (
	"context"
	"regexp"
	"strings"
)

// QConfigs returns the all configs from the OMNeT++ project.
func (project *OmnetProject) QConfigs(ctx context.Context) (configs []string, err error) {

	sim, err := project.command(ctx, "-s", "-a")
	if err != nil {
		return
	}

	byt, err := sim.CombinedOutput()
	if err != nil {
		return
	}

	output := string(byt)
	output = strings.TrimSpace(output)

	reg := regexp.MustCompile(`Config (.+?):`)
	matches := reg.FindAllStringSubmatch(output, -1)

	for _, match := range matches {
		configs = append(configs, match[1])
	}

	return
}

// QRunNumbers returns all runnumbers for the given config.
func (project *OmnetProject) QRunNumbers(ctx context.Context, config string) (numbers []string, err error) {

	//
	// Get runnumbers
	//

	sim, err := project.command(ctx, "-c", config, "-q", "runnumbers", "-s")
	if err != nil {
		return
	}

	byt, err := sim.CombinedOutput()
	if err != nil {
		return
	}

	output := string(byt)
	output = strings.TrimSpace(output)
	numbers = strings.Split(output, " ")

	return
}
