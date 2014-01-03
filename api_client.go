package wow

import (
	"fmt"
	"strings"
	"errors"
	"net/http"
	"net/url"
	"time"
	"encoding/base64"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/json"
	"io/ioutil"
	"strconv"
)

type ApiClient struct {
	Host string
	Locale string
	Secret string
	PublicKey string
}

func NewApiClient(region string, locale string) (*ApiClient, error) {
	var host string
	var validLocales []string
	switch region {
	case "US", "United States":
		host = "us.battle.net"
		validLocales = []string{"en_US", "es_MX", "pt_BR"}
	case "EU", "Europe":
		host = "eu.battle.net"
		validLocales = []string{"en_GB", "es_ES", "fr_FR", "ru_RU", "de_DE", "pt_PT", "it_IT"}
	case "KR", "Korea":
		host = "kr.battle.net"
		validLocales = []string{"ko_KR"}
	case "TW", "Taiwan":
		host = "tw.battle.net"
		validLocales = []string{"zh_TW"}
	case "ZH", "CN", "China":
		host = "www.battle.com.cn"
		validLocales = []string{"zh_CN"}
	default:
		return nil, errors.New(fmt.Sprintf("Region '%s' is not valid", region))
	}

	if locale == "" {
		return &ApiClient{Host: host, Locale: validLocales[0]}, nil
	} else {
		for _, valid := range validLocales {
			if valid == locale {
				return &ApiClient{Host: host, Locale: locale}, nil
			}
		}
	}
		
	return nil, errors.New(fmt.Sprintf("Locale '%s' is not valid for region '%s'", locale, region))
}

func (a *ApiClient) GetAchievement(id int) (*Achievement, error) {
	jsonBlob, err := a.get(fmt.Sprintf("achievement/%d", id))
	if err != nil {
		return nil, err
	}
	achieve := &Achievement{}
	err = json.Unmarshal(jsonBlob, achieve)
	if err != nil {
		return nil, err
	}
	return achieve, nil
}

func (a *ApiClient) GetAuctionData(realm string) (*AuctionData, error) {
	jsonBlob, err := a.get(fmt.Sprintf("auction/data/%s", realm))
	if err != nil {
		return nil, err
	}
	auctionData := &AuctionData{}
	err = json.Unmarshal(jsonBlob, auctionData)
	if err != nil {
		return nil, err
	}
	return auctionData, nil
}

func (a *ApiClient) GetBattlePetAbility(id int) (*BattlePetAbility, error) {
	jsonBlob, err := a.get(fmt.Sprintf("battlePet/ability/%d", id))
	if err != nil {
		return nil, err
	}
	ability := &BattlePetAbility{}
	err = json.Unmarshal(jsonBlob, ability)
	if err != nil {
		return nil, err
	}
	return ability, nil
}

func (a *ApiClient) GetBattlePetSpecies(id int) (*BattlePetSpecies, error) {
	jsonBlob, err := a.get(fmt.Sprintf("battlePet/species/%d", id))
	if err != nil {
		return nil, err
	}
	species := &BattlePetSpecies{}
	err = json.Unmarshal(jsonBlob, species)
	if err != nil {
		return nil, err
	}
	return species, nil
}

func (a *ApiClient) GetBattlePet(id int, level int, breedId int, qualityId int) (*BattlePet, error) {
	jsonBlob, err := a.getWithParams(
		fmt.Sprintf("battlePet/stats/%d", id), 
		map[string]string{
			"level": strconv.Itoa(level),
			"breedId": strconv.Itoa(breedId),
			"qualityId": strconv.Itoa(qualityId),
		})
	if err != nil {
		return nil, err
	}

	pet := &BattlePet{}
	err = json.Unmarshal(jsonBlob, pet)
	if err != nil {
		return nil, err
	}
	return pet, nil	
}

func (a *ApiClient) GetBattlePetStats(id int, level int, breedId int, qualityId int) (*BattlePet, error) {
	return a.GetBattlePet(id, level, breedId, qualityId)
}

// Will return region challenges if realm is empty string.
func (a *ApiClient) GetChallenges(realm string) ([]*Challenge, error) {
	if realm == "" {
		realm = "region"
	}
	jsonBlob, err := a.get(fmt.Sprintf("challenge/%s", realm))
	if err != nil {
		return nil, err
	}
	challengeSet := &challengeSet{}
	err = json.Unmarshal(jsonBlob, challengeSet)
	if err != nil {
		return nil, err
	}
	return challengeSet.Challenges, nil
}

func (a *ApiClient) GetChallenge(realm string) ([]*Challenge, error) {
	return a.GetChallenges(realm)
}

func (a *ApiClient) GetCharacter(realm string, characterName string) (*Character, error) {
	return a.GetCharacterWithFields(realm, characterName, make([]string, 0))
}

func (a *ApiClient) GetCharacterWithFields(realm string, characterName string, fields []string) (*Character, error) {
	err := validateCharacterFields(fields)
	if err != nil {
		return nil, err
	}
	jsonBlob, err := a.getWithParams(fmt.Sprintf("character/%s/%s", realm, characterName), map[string]string{"fields": strings.Join(fields, ",")})
	if err != nil {
		return nil, err
	}
	char := &Character{}
	err = json.Unmarshal(jsonBlob, char)
	if err != nil {
		return nil, err
	}
	return char, nil	
}

func validateCharacterFields(fields []string) error {
	badFields := make([]string, 0)
	for _, field := range fields {
		switch field {
			case "achievements",
			"appearance",
			"feed",
			"guild",
			"hunterPets",
			"items",
			"mounts",
			"pets",
			"petSlots",
			"professions",
			"progression",
			"pvp",
			"quests",
			"reputation",
			"stats",
			"talents",
			"titles":
			// valid, noop
		default:
			badFields = append(badFields, field)
		}
	}
	if len(badFields) != 0 {
		return errors.New(fmt.Sprintf("The following fields are not valid: %v", badFields))
	} else {
		return nil
	}

}

func (a *ApiClient) get(path string) ([]byte, error) {
	return a.getWithParams(path, make(map[string]string))
}

func (a *ApiClient) getWithParams(path string, queryParams map[string]string) ([]byte, error) {
	url := a.url(path, queryParams)
	client := &http.Client{}

	request, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return make([]byte, 0), err
	}

	if len(a.Secret) > 0 {
		request.Header.Add("Authorization", a.authorizationString(a.signature("GET", path)))
	}

	response, err := client.Do(request)
	if err != nil {
		return make([]byte, 0), err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return make([]byte, 0), err
	}
	
	return body, nil
}

func (a *ApiClient) url(path string, queryParamPairs map[string]string) *url.URL {
	queryParamPairs["locale"] = a.Locale
	queryParamList := make([]string, 0)
	for k, v := range queryParamPairs {
		queryParamList = append(queryParamList, k + "=" + v)
	}
	return &url.URL{
		Scheme: "http",
		Host: a.Host,
		Path: "/api/wow/" + path,
		RawQuery: strings.Join(queryParamList, "&"),
	}
}

func (a *ApiClient) authorizationString(signature string) string {
	return fmt.Sprintf(" BNET %s:%s", a.PublicKey, signature)
}

func (a *ApiClient) signature(verb string, path string) string {
	url := a.url(path, make(map[string]string))
	toBeSigned := []byte(strings.Join([]string{verb, time.Now().String(), url.Path, ""}, "\n"))
	mac := hmac.New(sha1.New, []byte(a.Secret))
	_, err := mac.Write(toBeSigned) // FIXME _ = signed
	if err != nil {
		handleError(err)
	}
	return base64.StdEncoding.EncodeToString([]byte("hi")) //FIXME Figure out crypto
}

func handleError(err error) {
	panic(err)
}

