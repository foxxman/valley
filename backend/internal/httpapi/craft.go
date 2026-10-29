package httpapi

import (
	"context"
	"net/http"

	"game/backend/internal/craft"
	"game/backend/internal/inventory"
)

type CraftService interface {
	List(ctx context.Context, gameID string) ([]craft.Listed, error)
	Craft(ctx context.Context, gameID, playerID, recipeSlug string) (inventory.Inventory, error)
}

type recipeListBody struct {
	Recipes []recipeBody `json:"recipes"`
} //@name recipeList

type recipeBody struct {
	Slug        string           `json:"slug" example:"chest"`
	Result      recipePartBody   `json:"result"`
	Ingredients []recipePartBody `json:"ingredients"`
} //@name recipe

type recipePartBody struct {
	Slug     string `json:"slug" example:"wood"`
	Name     string `json:"name" example:"Древесина"`
	IconKey  string `json:"iconKey" example:"wood"`
	Quantity int    `json:"quantity" example:"4"`
	MaxStack int    `json:"maxStack" example:"99"`
} //@name recipePart

type craftRequest struct {
	Recipe string `json:"recipe" example:"chest"`
} //@name craftRequest

func RegisterCraft(mux *http.ServeMux, recipes CraftService, sessions PlayerSession) {
	h := &craftHandler{recipes: recipes, sessions: sessions}
	mux.HandleFunc("GET /api/games/{id}/recipes", h.list)
	mux.HandleFunc("POST /api/games/{id}/actions/craft", h.craft)
}

type craftHandler struct {
	recipes  CraftService
	sessions PlayerSession
}

// list возвращает рецепты, которые можно создать из рюкзака.
//
//	@Summary	Список рецептов
//	@Tags		craft
//	@Produce	json
//	@Security	AccessCookie
//	@Param		id	path		string	true	"ID игры"
//	@Success	200	{object}	recipeListBody
//	@Failure	401	{object}	errorBody	"unauthorized"
//	@Failure	404	{object}	errorBody	"game_not_found"
//	@Failure	500	{object}	errorBody
//	@Router		/api/games/{id}/recipes [get]
func (h *craftHandler) list(w http.ResponseWriter, r *http.Request) {
	if _, ok := authenticate(w, r, h.sessions); !ok {
		return
	}
	listed, err := h.recipes.List(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toRecipeList(listed))
}

// craft списывает ингредиенты и кладёт результат в рюкзак.
//
//	@Summary	Создать по рецепту
//	@Tags		craft
//	@Accept		json
//	@Produce	json
//	@Security	AccessCookie
//	@Param		id		path		string			true	"ID игры"
//	@Param		body	body		craftRequest	true	"Код рецепта"
//	@Success	200		{object}	inventoryBody
//	@Failure	400		{object}	errorBody	"invalid_body или unknown_recipe"
//	@Failure	401		{object}	errorBody	"unauthorized"
//	@Failure	404		{object}	errorBody	"game_not_found"
//	@Failure	409		{object}	errorBody	"missing_ingredients или bag_full"
//	@Failure	500		{object}	errorBody
//	@Router		/api/games/{id}/actions/craft [post]
func (h *craftHandler) craft(w http.ResponseWriter, r *http.Request) {
	current, ok := authenticate(w, r, h.sessions)
	if !ok {
		return
	}
	var body craftRequest
	if err := decodeBody(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	bag, err := h.recipes.Craft(r.Context(), r.PathValue("id"), current.ID, body.Recipe)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toInventoryBody(bag))
}

func toRecipeList(listed []craft.Listed) recipeListBody {
	body := recipeListBody{Recipes: make([]recipeBody, 0, len(listed))}
	for _, recipe := range listed {
		ingredients := make([]recipePartBody, 0, len(recipe.Ingredients))
		for _, part := range recipe.Ingredients {
			ingredients = append(ingredients, toRecipePart(part))
		}
		body.Recipes = append(body.Recipes, recipeBody{
			Slug:        recipe.Slug,
			Result:      toRecipePart(recipe.Result),
			Ingredients: ingredients,
		})
	}
	return body
}

func toRecipePart(part craft.Part) recipePartBody {
	return recipePartBody{
		Slug:     part.Item.Slug,
		Name:     part.Item.Name,
		IconKey:  part.Item.IconKey,
		Quantity: part.Quantity,
		MaxStack: part.Item.MaxStack,
	}
}
