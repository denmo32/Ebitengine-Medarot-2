package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func parseInt(s string, defaultValue int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return i
}

func parseBool(s string) bool {
	return strings.ToLower(strings.TrimSpace(s)) == "true"
}

func LoadMedals(filePath string) ([]Medal, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Read() // Skip header

	var medals []Medal
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		medals = append(medals, Medal{
			ID:         record[0],
			Name:       record[1],
			SkillLevel: parseInt(record[2], 1),
		})
	}
	return medals, nil
}

func LoadParts(filePath string) (map[string]*Part, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Read() // Skip header

	partsMap := make(map[string]*Part)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 12 {
			continue
		}
		armor := parseInt(record[4], 1)
		part := &Part{
			ID:         record[0],
			PartName:   record[1],
			Type:       PartType(record[2]),
			Category:   PartCategory(record[3]),
			Armor:      armor,
			MaxArmor:   armor,
			Power:      parseInt(record[5], 0),
			Charge:     parseInt(record[6], 1),
			Cooldown:   parseInt(record[7], 1),
			Accuracy:   parseInt(record[8], 0),
			Mobility:   parseInt(record[9], 0),
			Propulsion: parseInt(record[10], 0),
			Trait:      Trait(record[11]),
			IsBroken:   false,
		}
		partsMap[part.ID] = part
	}
	return partsMap, nil
}

func LoadMedarotLoadouts(filePath string) ([]MedarotData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Read() // Skip header

	var medarots []MedarotData
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 10 {
			continue
		}
		medarot := MedarotData{
			ID:         record[0],
			Name:       record[1],
			Team:       TeamID(parseInt(record[2], 0)),
			IsLeader:   parseBool(record[3]),
			DrawIndex:  parseInt(record[4], 0),
			MedalID:    record[5],
			HeadID:     record[6],
			RightArmID: record[7],
			LeftArmID:  record[8],
			LegsID:     record[9],
		}
		medarots = append(medarots, medarot)
	}
	return medarots, nil
}

func LoadAllGameData() (*GameData, error) {
	gameData := &GameData{}
	var err error

	gameData.Medals, err = LoadMedals("data/medals.csv")
	if err != nil {
		return nil, fmt.Errorf("medals.csvの読み込みに失敗: %w", err)
	}

	gameData.AllParts, err = LoadParts("data/parts.csv")
	if err != nil {
		return nil, fmt.Errorf("parts.csvの読み込みに失敗: %w", err)
	}

	gameData.Medarots, err = LoadMedarotLoadouts("data/medarots.csv")
	if err != nil {
		return nil, fmt.Errorf("medarots.csvの読み込みに失敗: %w", err)
	}

	return gameData, nil
}
