"""
This script is for performing basic analysis on the NBA game data with pregame and in-game odds,
using the two csvs populated by the game and odds data collection and processing jobs. To configure
pregame and/or in game filters, fill in the fields in AnalysisConfig.py. Empty filters will be ignored. 
The hardcoded CSV definitions are below

Game Data CSV Column Indices:
0: game_id
1: season_id
2: game_date
3: start_time
4: away_team_init
5: away_team_id
6: home_team_init
7: home_team_id
8: away_ml
9: away_spread
10: home_ml
11: home_spread
12: pregame_total
13: away_final_score
14: home_final_score

Play By Play CSV Column Indices:
0: game_id
1: seconds_elapsed
2: away_score
3: home_score
4: underdog_score
5: favorite_score
6: favorite_margin
"""

import analysis_config as cfg
from enum import Enum
import csv
import numpy as np 


# CSV_DIRECTORY = '../csvs'
CSV_DIRECTORY = '/Users/ericwhitehead/Desktop/clag/nba-project-post-mv/csvs'
GAME_SUMMARY_CSV = 'games_summary_data.csv'
PLAY_BY_PLAY_CSV = 'game_play_by_play_data.csv'

class AnalysisType(Enum):
    INGAME = 1
    PREGAME = 2

class FilterType(Enum):
    EQUALITY = 1
    IN_RANGE = 2

class FilterField(Enum):
    def __init__(self, csvIndex, value, filterType):
        self._csvIndex = csvIndex
        self._filterType = filterType
        self._value = value
        
    @property
    def csvIndex(self):
        return self._csvIndex

    @property 
    def filterType(self):
        return self._filterType

    @property
    def value(self):
        return self._value 
    

# Game Summary CSV row lookups 
def getUnderdogId(row):
    return row[5] if float(row[10]) > 0 else row[7]

def getFavoriteId(row):
    return row[5] if float(row[10]) <= 0 else row[7]

def getFavoriteSpread(row):
    return abs(float(row[10]))

def getFavoriteMoneyline(row):
    return float(row[8]) if float(row[10]) <= 0 else float(row[9])

# Play by Play CSV row lookups
def getFavoriteMargin(playsRow):
    return int(playsRow[5]) - int(playsRow[6])

def getTotal(playsRow):
    return int(playsRow[5]) + int(playsRow[6])


class GameFilterFields(FilterField):
    E_AWAY_TEAM_IDS = (lambda row: row[5], cfg.AWAY_TEAM_IDS, FilterType.EQUALITY)
    E_HOME_TEAM_IDS = (lambda row: row[7], cfg.HOME_TEAM_IDS, FilterType.EQUALITY) 
    E_UNDERDOG_TEAM_IDS = (getUnderdogId, cfg.UNDERDOG_TEAM_IDS, FilterType.EQUALITY)
    E_FAVORITE_TEAM_IDS = (getFavoriteId, cfg.FAVORITE_TEAM_IDS, FilterType.EQUALITY)
    E_FAVORITE_SPREAD_RANGE = (getFavoriteSpread, cfg.PREGAME_FAVORITE_SPREAD_RANGE, FilterType.IN_RANGE)
    E_FAVORITE_ML_RANGE = (getFavoriteMoneyline, cfg.PREGAME_FAVORITE_ML_RANGE, FilterType.IN_RANGE)
    E_TOTAL_RANGE = (lambda row: float(row[12]), cfg.PREGAME_TOTAL_RANGE, FilterType.IN_RANGE)
    E_MONTHS = (lambda row: int(row[2][5:7]), cfg.MONTHS, FilterType.EQUALITY)
    E_SEASONS = (lambda row: row[1], cfg.SEASONS, FilterType.EQUALITY)

class PlayByPlayFilterFields(FilterField): 
    E_IN_GAME_ELAPSED_SECONDS_RANGE = (lambda row: int(row[1]), cfg.IN_GAME_ELAPSED_SECONDS_RANGE, FilterType.IN_RANGE)
    E_IN_GAME_FAVORITE_MARGIN_RANGE = (getFavoriteMargin, cfg.IN_GAME_FAVORITE_MARGIN_RANGE, FilterType.IN_RANGE)
    E_IN_GAME_TOTAL_RANGE = (getTotal, cfg.IN_GAME_TOTAL_RANGE, FilterType.IN_RANGE)
    

def checkCriteria(row, valueGetter, filterValues, filterType):
    if filterValues == []:
        return True
    
    val = valueGetter(row)
    if filterType == FilterType.EQUALITY:
        return val in filterValues
    else:
        lo, hi = filterValues
        return val >= lo and val <= hi
        
        
def loadCsvAndFilter(csvName, filterEnum):
    filteredRows = []

    with open(f'{CSV_DIRECTORY}/{csvName}', mode ='r') as file:
        rows = csv.reader(file)
        
        next(rows)
        for row in rows:
            if all(checkCriteria(row, f.csvIndex, f.value, f.filterType) for f in filterEnum):
                filteredRows.append(row)

    return filteredRows

def printFilters():
    print("PREGAME FILTER FIELDS:")
    for filt in GameFilterFields:
        if filt.value:
            print(f'{filt.name[2:]}={filt.value}')

    print("\nINGAME FILTER FIELDS:")
    for filt in PlayByPlayFilterFields:
        if filt.value:
            print(f'{filt.name[2:]}={filt.value}')
    

def printStatistics(data):
    print(f'Sample size: {len(data)} games \n')
    if len(data) > 0:
        print(f'Average: {round(np.average(data), 2)} pts')
        print('Deciles -  ')
        for dec in [10, 20, 30, 40, 50, 60, 70, 80, 90]:
            print(f'{dec}% {round(np.percentile(data, dec), 2)} pts')


def processResults(gameCsvRows):
    totals = []
    favoriteMargins = []

    for game in gameCsvRows:
        favScore = int(game[13]) if float(game[10]) <= 0 else int(game[14])
        dogScore = int(game[14]) if float(game[10]) < 0 else int(game[13])
        total = favScore + dogScore
        totals.append(total)
        favoriteMargins.append(favScore - dogScore)

    printFilters()

    print('\nENDGAME TOTALS ANALYSIS') 
    printStatistics(totals)

    print('\nENDGAME FAVORITE MARGINS ANALYSIS')
    printStatistics(favoriteMargins)


if __name__ == '__main__':
    analysisType = AnalysisType.INGAME
    gameCsvRows = loadCsvAndFilter(GAME_SUMMARY_CSV, GameFilterFields)

    if analysisType == AnalysisType.INGAME:
        playsCsvGameIds = set(map(lambda row: row[0], loadCsvAndFilter(PLAY_BY_PLAY_CSV, PlayByPlayFilterFields))) 
        gameCsvRows = list(filter(lambda row: row[0] in playsCsvGameIds, gameCsvRows))

    processResults(gameCsvRows)
