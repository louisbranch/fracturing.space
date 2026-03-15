package workflow

import (
	"net/url"
	"testing"
)

func TestInstallResolvesNormalizedIDsAndAliases(t *testing.T) {
	t.Parallel()

	workflow := registryWorkflow{}
	registry := Install(
		Installation{
			ID:                " daggerheart ",
			Aliases:           []string{"DAGGERHEART", "game_system_daggerheart"},
			CharacterCreation: workflow,
		},
		Installation{
			ID:      "ignored",
			Aliases: []string{"unused"},
		},
	)

	for _, system := range []string{"daggerheart", "DAGGERHEART", " game_system_daggerheart "} {
		if got := registry.Resolve(system); got == nil {
			t.Fatalf("Resolve(%q) = nil, want installed workflow", system)
		}
	}
	if got := registry.Resolve("unknown"); got != nil {
		t.Fatalf("Resolve(unknown) = %#v, want nil", got)
	}
}

func TestInstallReturnsNilWhenNoUsableWorkflowsExist(t *testing.T) {
	t.Parallel()

	registry := Install(
		Installation{},
		Installation{ID: "daggerheart"},
	)
	if registry != nil {
		t.Fatalf("Install() = %#v, want nil", registry)
	}
}

func TestServicesEnabledReflectInstalledWorkflowRegistry(t *testing.T) {
	t.Parallel()

	registry := Install(Installation{ID: "daggerheart", CharacterCreation: registryWorkflow{}})

	page := NewPageService(&workflowAppStub{}, registry)
	if !page.Enabled("DAGGERHEART") {
		t.Fatalf("PageService.Enabled(DAGGERHEART) = false, want true")
	}
	if page.Enabled("unknown") {
		t.Fatalf("PageService.Enabled(unknown) = true, want false")
	}

	mutation := NewMutationService(&workflowAppStub{}, registry)
	if !mutation.Enabled("daggerheart") {
		t.Fatalf("MutationService.Enabled(daggerheart) = false, want true")
	}
	if mutation.Enabled("unknown") {
		t.Fatalf("MutationService.Enabled(unknown) = true, want false")
	}
}

type registryWorkflow struct {
	view      CharacterCreationView
	parseForm url.Values
}

func (t registryWorkflow) BuildView(progress Progress, catalog Catalog, profile Profile) CharacterCreationView {
	return CharacterCreationView{
		Ready:    progress.Ready,
		NextStep: progress.NextStep,
	}
}

func (t registryWorkflow) ParseStepInput(form url.Values, nextStep int32) (*StepInput, error) {
	return &StepInput{}, nil
}
