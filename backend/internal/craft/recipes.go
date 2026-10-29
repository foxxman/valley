package craft

// Ingredient is one line of a recipe cost.
type Ingredient struct {
	Slug     string
	Quantity int
}

// Recipe is a server-owned crafting definition. Item names live in the catalog.
type Recipe struct {
	Slug           string
	ResultSlug     string
	ResultQuantity int
	Ingredients    []Ingredient
}

func Recipes() []Recipe {
	return []Recipe{
		{
			Slug:           "chest",
			ResultSlug:     "chest",
			ResultQuantity: 1,
			Ingredients: []Ingredient{
				{Slug: "wood", Quantity: 4},
			},
		},
	}
}

func Find(slug string) (Recipe, bool) {
	for _, recipe := range Recipes() {
		if recipe.Slug == slug {
			return recipe, true
		}
	}
	return Recipe{}, false
}
