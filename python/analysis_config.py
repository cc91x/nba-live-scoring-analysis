""" Contains filter parameters for our data analysis """

# Pregame Filters. By default, leaving blank [] will apply no filter for each filer type.

# Team IDs, as strings, ex. ['1610612739', '1610612739']
AWAY_TEAM_IDS = []
HOME_TEAM_IDS = []
UNDERDOG_TEAM_IDS = []
FAVORITE_TEAM_IDS = []

# Months, as integers
MONTHS = []

# Season IDs, as strings. ex. ['22024'] for regular season 2024
SEASONS = []

# Pregame ranges for odds types. Should be two numbers greater than 0, making a range [min, max].
PREGAME_FAVORITE_SPREAD_RANGE = []
PREGAME_FAVORITE_ML_RANGE = []
PREGAME_TOTAL_RANGE = []
 
# In game filters, used in conjunction with each other. Filling all three would apply a filter such as:
#   "Games where the favorite led by 5-8 pts in mins 3-5 and the total is between 5 and 15" etc.
IN_GAME_ELAPSED_SECONDS_RANGE = []
IN_GAME_FAVORITE_MARGIN_RANGE = []
IN_GAME_TOTAL_RANGE = []
