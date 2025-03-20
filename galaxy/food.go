package galaxy

import "galaxy.io/server/galaxy/utils"

// Food represents an alive food item in a game.
type Food struct {
	position utils.Vector2D
	color    uint32
}
