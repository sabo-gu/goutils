package bp_nn

import (
	deep "github.com/patrikeh/go-deep"
	"github.com/patrikeh/go-deep/training"
)

type NNModel struct {
	Inputs            int
	Layout            []int
	Activation        deep.ActivationType
	Mode              deep.Mode // ModeBinary
	UseBias           bool
	WeightInitializer deep.WeightInitializer
	// private
	dataset training.Examples
}

func NewNNModel(inputs int, layout []int, activation deep.ActivationType, mode deep.Mode, useBias bool, initializer deep.WeightInitializer) *NNModel {

	return &NNModel{
		Inputs:            inputs,
		Layout:            layout,
		Activation:        activation,
		Mode:              mode,
		UseBias:           useBias,
		WeightInitializer: initializer,
		dataset:           make([]training.Example, 0),
	}
}

func (nn *NNModel) AddSamples(example ...training.Example) {
	nn.dataset = append(nn.dataset, example...)

}

func (nn *NNModel) Train() {
	config := &deep.Config{
		/* Input dimensionality */
		Inputs: nn.Inputs,
		/* Two hidden layers consisting of two neurons each, and a single output */
		Layout: nn.Layout,
		/* Activation functions: Sigmoid, Tanh, ReLU, Linear */
		Activation: nn.Activation,
		/* Determines output layer activation & loss function:
		ModeRegression: linear outputs with MSE loss
		ModeMultiClass: softmax output with Cross Entropy loss
		ModeMultiLabel: sigmoid output with Cross Entropy loss
		ModeBinary: sigmoid output with binary CE loss */
		Mode: nn.Mode,
		/* Weight initializers: {deep.NewNormal(μ, σ), deep.NewUniform(μ, σ)} */
		Weight: nn.WeightInitializer,
		/* Apply bias */
		Bias: nn.UseBias,
	}

	network := deep.NewNeural(config)
	// Solver
	// optimizer := training.NewSGD(0.05, 0.1, 1e-6, true)
	optimizer := training.NewAdam(0.001, 0.9, 0.999, 1e-8)
	trainer := training.NewTrainer(optimizer, 50)

	trainingSet, validSet := nn.dataset.Split(0.8)
	trainer.Train(network, trainingSet, validSet, 1000)

}

func Test() {

}
