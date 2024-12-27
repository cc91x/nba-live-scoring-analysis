from argparse import ArgumentParser 
from datetime import datetime, timedelta
from enum import Enum
from pymongo import MongoClient
import logging

from nba_api.stats.endpoints import teamgamelog
from nba_api.stats.endpoints import playbyplay
import sourcing_config as cfg

DATE_STRING_FORMAT = '%Y-%m-%d'

# Classes and enums 
class MongoNamingInfo:
    RAW_GAME_COLLECTION = 'rawGames'
    TEAM_ID_COLLECTION = 'teamMetadata'

    TEAM_ID_FIELD = 'teamId'
    TEAM_ABBREV_FIELD = 'teamAbbreviation'

    DATE_FIELD = 'date'
    GAME_ID_FIELD = 'gameId'
    MATCHUP_FIELD = 'matchup'
    SEASON_ID = 'seasonId'
    RAW_PLAY_BY_PLAY_FIELD = 'rawPlayByPlay' 


class SeasonType(Enum):
    def __init__(self, id, name):
        self._id = id
        self._name = name
        
    @property
    def id(self):
        return self._id

    @property 
    def name(self):
        return self._name
    
    REGULAR_SEASON = ('2' + cfg.SeasonInfo.SEASON_ID, 'Regular Season')
    PLAYOFFS = ('4' + cfg.SeasonInfo.SEASON_ID, 'Playoffs')


class GamelogData():
    def __init__(self, row):
        game_id, date_string, matchup = row[1], row[2], row[3]
        self._date = datetime.strptime(date_string, '%b %d, %Y').strftime(DATE_STRING_FORMAT)
        self._game_id = game_id
        self._matchup = matchup

    @property
    def date(self):
        return self._date
    
    @property
    def game_id(self):
        return self._game_id

    @property
    def matchup(self):
        return self._matchup

# Miscellaneous functions
def extract_date_parameter():
    parser = ArgumentParser()
    parser.add_argument('date', type=str)
    date_str = parser.parse_args().date

    datetime_obj = datetime.strptime(date_str, DATE_STRING_FORMAT)
    prev_date = datetime_obj - timedelta(days=2) 
    return prev_date.strftime(DATE_STRING_FORMAT)


def get_season_type_from_date(date): 
    return SeasonType.REGULAR_SEASON if date < cfg.SeasonInfo.PLAYOFF_START_DATE else SeasonType.PLAYOFFS


def load_team_id_map(mongo_db):
    team_id_collection = mongo_db[MongoNamingInfo.TEAM_ID_COLLECTION]
    team_map = {d[MongoNamingInfo.TEAM_ID_FIELD]: d[MongoNamingInfo.TEAM_ABBREV_FIELD] for d in team_id_collection.find()}
    return team_map


def find_gamelogs_for_date(team_ids, date_parameter, season_year, season_type):
    gamelogs = {}
    for team_id in team_ids:
        season = teamgamelog.TeamGameLog(str(team_id), season=season_year, season_type_all_star=season_type.name)

        for row in season.get_dict()['resultSets'][0]['rowSet']:
            gamelog = GamelogData(row)
            if gamelog.date == date_parameter and gamelog.game_id not in gamelogs:
                gamelogs[gamelog.game_id] = gamelog

    logger.info(f'Found {len(gamelogs)} games')
    return gamelogs


def add_gamelog_fields_to_game(raw_game_dict, gamelog, season_id):     
    raw_game_dict[MongoNamingInfo.DATE_FIELD] = gamelog.date
    raw_game_dict[MongoNamingInfo.GAME_ID_FIELD] = gamelog.game_id
    raw_game_dict[MongoNamingInfo.MATCHUP_FIELD] = gamelog.matchup
    raw_game_dict[MongoNamingInfo.SEASON_ID] = season_id  


def extract_raw_play_by_play(raw_game_dict):
    play_by_play_obj = [val for i, val in enumerate(raw_game_dict['resultSets']) if val.get('name', '') == 'PlayByPlay']
    play_by_play_rows = play_by_play_obj[0]['rowSet']
    raw_game_dict[MongoNamingInfo.RAW_PLAY_BY_PLAY_FIELD] = play_by_play_rows


def get_playbyplay_and_save(gamelogs, mongo_db, season_id):
    raw_games_collection = mongo_db[MongoNamingInfo.RAW_GAME_COLLECTION]
    games_to_insert = []
    for game_id, gamelog in gamelogs.items():      
        if raw_games_collection.find_one({MongoNamingInfo.GAME_ID_FIELD: game_id}) == None:
            
            playbyplay_response = playbyplay.PlayByPlay(game_id=game_id)
            raw_game_dict =  playbyplay_response.get_dict()

            extract_raw_play_by_play(raw_game_dict)
            add_gamelog_fields_to_game(raw_game_dict, gamelog, season_id)
            
            games_to_insert.append(raw_game_dict)

    new_games, existing_games = len(games_to_insert), len(gamelogs.items()) - len(games_to_insert) 
    logger.info(f'Found {existing_games} existing games. Inserting {new_games} new games.')
    
    if new_games:
        raw_games_collection.insert_many(games_to_insert)     
 

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(message)s')
    logger = logging.getLogger(__name__)
    
    mongo_client = MongoClient(cfg.MongoConfig.MONGO_CONNECTION_STRING)
    mongo_db = mongo_client[cfg.MongoConfig.MONGO_DB_NAME]

    date_parameter = extract_date_parameter()
    season_type = get_season_type_from_date(date_parameter)

    logger.info(f'Beginning game lookup for {date_parameter}, season type: {season_type.name}')

    team_map = load_team_id_map(mongo_db)    
    gamelogs = find_gamelogs_for_date(team_map.keys(), date_parameter, cfg.SeasonInfo.SEASON_YEAR, season_type)
    get_playbyplay_and_save(gamelogs, mongo_db, season_type.id)
