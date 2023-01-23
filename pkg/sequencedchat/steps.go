package sequencedchat

import (
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
)

type StepsHandler struct {
	StaticLogic        map[string]map[string]string
	Steps              map[string]LogicStep
	DynamicStepActions func(c *Chat, userInput string, bot *tgbotapi.BotAPI) (doOverride bool, newStepName string, overrideText string)
	DeveloperErrorText string
	WhenNotFoundText   string
}

func (m StepsHandler) getStep(stepName string) (LogicStep, bool) {
	step, ok := m.Steps[stepName]
	if !ok {
		log.Error().Msgf("got a strange non-existent step name: %s", stepName)
		return LogicStep{}, false
	}
	step.Id = stepName
	return step, true
}

func (m StepsHandler) GenerateAnswer(c *Chat, userInput string, bot *tgbotapi.BotAPI) LogicStep {
	log.Debug().Msgf("New request. prevStep: %q, userInput: %q", c.CurrentStage, userInput)

	found, step := m.getCommandsStep(c.CurrentStage, userInput)
	if found {
		return step
	}

	override, step := m.getNextDynamicStep(c, userInput, bot)
	if override {
		return step
	}

	return m.getNextStaticStep(c.CurrentStage, userInput)
}

func (m StepsHandler) getCommandsStep(currentStep, userInput string) (bool, LogicStep) {
	firstChar, _ := utf8.DecodeRuneInString(userInput)
	if firstChar == '/' {
		commandsList := m.StaticLogic[""] // All commands are stored under this first blank entry
		stepName, commandExist := commandsList[userInput]
		if !commandExist {
			return true, LogicStep{
				Id:   currentStep,
				Text: m.WhenNotFoundText,
			}
		}
		step, ok := m.getStep(stepName)
		if !ok {
			return true, m.developerError()
		}
		return true, step
	}
	return false, LogicStep{}
}

func (m StepsHandler) getNextDynamicStep(c *Chat, userInput string, bot *tgbotapi.BotAPI) (bool, LogicStep) {
	prevStepName := c.CurrentStage
	if prevStepName != "" {
		prevStep, ok := m.getStep(prevStepName)
		if !ok {
			return true, m.developerError()
		}
		// We can dynamically alter the step logic. In case doOverride is false, we'll use the static logic from 'staticLogic' variable
		doOverride, overrideNextStep, overrideText := m.DynamicStepActions(c, userInput, bot)
		if doOverride {
			if overrideNextStep == prevStepName {
				if overrideText != "" {
					prevStep.Text = overrideText
				}
				return true, prevStep
			}
			nextStep, ok := m.getStep(overrideNextStep)
			if !ok {
				return true, m.developerError()
			}
			if overrideText != "" {
				nextStep.Text = overrideText
			}
			return true, nextStep
		}
	}
	return false, LogicStep{}
}

func (m StepsHandler) getNextStaticStep(prevStepName, userInput string) LogicStep {
	// sanity check
	_, ok := m.StaticLogic[prevStepName]
	if !ok {
		log.Error().Msgf("we got a strange non-existent step name: %s", prevStepName)
		return m.developerError()
	}

	stepName, ok := m.StaticLogic[prevStepName][userInput]
	if !ok {
		stepName, ok = m.StaticLogic[prevStepName]["any"]
		if !ok {
			return LogicStep{
				Id:   prevStepName,
				Text: m.WhenNotFoundText,
			}
		}
	}
	answer, ok := m.getStep(stepName)
	if !ok {
		log.Trace().Msgf("developer forgot to add step %q into m.Steps map", stepName)
		return m.developerError()
	}
	return answer
}

func (m StepsHandler) developerError() LogicStep {
	return LogicStep{
		Text: m.DeveloperErrorText,
	}
}
