package categories

import (
	"container/list"
	"dimi/ml-auth/requests"
	"encoding/json"
	"fmt"
	"os"
)

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListingPrice struct {
	ListingTypeID string  `json:"listing_type_id"`
	SaleFeeAmount float64 `json:"sale_fee_amount"`
	CurrencyID    string  `json:"currency_id"`
}

type SubCategory struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	ChildrenCategories []Category `json:"children_categories"`
}

func GetCategories() ([]Category, error) {
	url := "https://api.mercadolibre.com/sites/MLB/categories"

	body, err := requests.MakeSimpleRequest(url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer requisição: %s", err)
	}

	result := []Category{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer unmarshal: %s", err)
	}

	return result, nil
}

func GetListingPrices(category string) ([]ListingPrice, error) {
	url := "https://api.mercadolibre.com/sites/MLB/listing_prices?price=100&category_id=" + category
	fmt.Println("URL:", url)

	res, err := requests.MakeRequest(url, nil)
	defer res.Body.Close()

	var prices []ListingPrice
	if err := json.NewDecoder(res.Body).Decode(&prices); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao fazer requisição: %s", err)
	}

	return prices, nil
}

func fetchCategory(categoryID string) (*SubCategory, error) {
	url := fmt.Sprintf("https://api.mercadolibre.com/categories/%s", categoryID)
	body, err := requests.MakeSimpleRequest(url, nil)
	if err != nil {
		return nil, err
	}

	var cat SubCategory
	err = json.Unmarshal(body, &cat)
	if err != nil {
		return nil, err
	}

	return &cat, nil
}

func PrintCategoryTree(categoryID string, indent string, l *list.List) {
	subcat, err := fetchCategory(categoryID)
	if err != nil {
		fmt.Printf("Erro ao buscar categoria %s: %v\n", categoryID, err)
		return
	}

	cat := Category{
		ID:   subcat.ID,
		Name: subcat.Name,
	}

	l.PushFront(cat)

	fmt.Printf("%s- %s (%s)\n", indent, subcat.Name, subcat.ID)

	for _, child := range subcat.ChildrenCategories {
		PrintCategoryTree(child.ID, indent+"  ", l)
	}
}

func GetAllCategories() ([]Category, error) {
	all_cat, err := GetCategories()
	if err != nil {
		fmt.Println("Erro ao conseguir categorias: ", err)
		return nil, err
	}

	l := list.New()

	for _, cat := range all_cat {
		PrintCategoryTree(cat.ID, "  ", l)
	}

	arr := make([]Category, l.Len())
	pos := 0
	for l := l.Front(); l != nil; l = l.Next() {
		arr[pos] = l.Value.(Category)
		pos++
	}
	return arr, nil
}

func SaveCategoriesToFile(categories []Category) error {
	data, err := json.MarshalIndent(categories, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar categorias: %v", err)
	}

	err = os.WriteFile("all_categories.json", data, 0644)
	if err != nil {
		return fmt.Errorf("erro ao escrever no arquivo: %v", err)
	}

	return nil
}

func LoadCategoriesFromFile() ([]Category, error) {
	data, err := os.ReadFile("all_categories.json")
	if err != nil {
		return nil, fmt.Errorf("erro ao ler o arquivo: %v", err)
	}

	var categories []Category
	err = json.Unmarshal(data, &categories)
	if err != nil {
		return nil, fmt.Errorf("erro ao deserializar categorias: %v", err)
	}

	return categories, nil
}

func LoadOrFetchCategories() ([]Category, error) {
	const fileName = "all_categories.json"

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		fmt.Println("Arquivo não encontrado. Buscando categorias...")
		categories, err := GetAllCategories()
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar categorias: %v", err)
		}

		err = SaveCategoriesToFile(categories)
		if err != nil {
			return nil, fmt.Errorf("erro ao salvar categorias: %v", err)
		}

		return categories, nil
	}

	fmt.Println("Arquivo encontrado. Carregando categorias do disco...")
	return LoadCategoriesFromFile()
}
