package galaxy

// Food represents an alive food item in a game.
type Food struct {
	position Vector2D
	color    uint32
}

func createRandomFood() []Food {
	// 100 comidas
	var food []Food
	for i := 0; i < 4000; i++ {
		food = append(food, Food{
			position: *randomPosition(),
			color: randomColor(),
		})
	}

	return food
}

func randomColor() uint32 {
	return 0x0
}
