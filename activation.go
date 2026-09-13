package main

type ActivationSettings struct {
	Model  string
	Prompt string
}

func activationSettingsFromConfig(cfg Config) ActivationSettings {
	return ActivationSettings{
		Model:  cfg.ActivationModel,
		Prompt: "hi",
	}
}
