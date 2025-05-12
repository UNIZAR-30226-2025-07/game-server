package galaxy

import "math/rand"

// Colors
const (
	Red    uint32 = 0xFF0000
	Green  uint32 = 0x00FF00
	Blue   uint32 = 0x0000FF
	Yellow uint32 = 0xFFFF00
	Cyan   uint32 = 0x00FFFF
	Magenta uint32 = 0xFF00FF
	Orange  uint32 = 0xFFA500
	Purple  uint32 = 0x800080
	Pink    uint32 = 0xFFC0CB
	Brown   uint32 = 0xA52A2A
	Lime    uint32 = 0x00FF00 // Same as bright green
	Teal    uint32 = 0x008080
	Navy    uint32 = 0x000080
	Maroon  uint32 = 0x800000
	Olive   uint32 = 0x808000
)
var FoodColors = []uint32{
	Red,
	Green,
	Blue,
	Yellow,
	Cyan,
	Magenta,
	Orange,
	Purple,
	Pink,
	Brown,
	Lime,
	Teal,
	Navy,
	Maroon,
	Olive,
}
// Food represents an alive food item in a game.
type Food struct {
	position Vector2D
	color    uint32
}

func createRandomFood() []Food {
	// 100 comidas
	var food []Food
	for i := 0; i < 25; i++ {
		food = append(food, Food{
			position: *randomPosition(),
			color: randomColor(),
		})
	}

	return food
}

func randomColor() uint32 {
	randomIndex := rand.Intn(len(FoodColors))
	return FoodColors[randomIndex]
}
