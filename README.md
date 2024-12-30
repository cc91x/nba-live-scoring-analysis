## Background

The purpose of this project is to examine the likelihood of an end game result based on a live in game scenario. I was inspired to create this project while watching the Celtics - and many other great teams - erase 2nd, 3rd and 4th quarter deficits and win games. And with the availability of live, in game NBA betting, I wondered just how likely are these types of comebacks really were, along with all other sorts of in game happenings. If a heavily favored team is down in the 1st, 2nd or 3rd quarter, what is the chance they come back and win? If a 10 point underdog has a lead going into the 4th, how often do they hang on and win? If the two teams combine for only 90 points in the first half, how likely is it they will hit the over? These are the types of questions the project seeks to address with historical data.

## Method

The NBA makes their statistics accessible via APIs. With this [python API client](https://github.com/swar/nba_api), we can find practically any piece of information related to the NBA, but for this project, we'll focus on game logs with play by play data. We can also source pregame NBA game odds from a different provider and link them with our individual play by play game data. We then save them as [csvs](csvs) for easier analysis - one for high level game summaries, and another with game scoring at 30 second intervals.

## Setup

The properties in the [config file](go/go_config.yaml) are needed for the go tasks. Input a MongoDB host, port, schema name, and Odds API key. 

### **MongoDB**

MongoDB is used to store the raw and processed data. Connect to a mongoDB instance that has these five collections defined:
* cleanedGameData
* cleanedOdds
* rawGames
* rawHistoricalOdds
* teamMetadata (Note: this collection needs to be populated before running anything. See [teamMetadata.json](mongodb/teamMetadata.json))

### **Golang** 

The golang package in the project needs to be compiled. From the [go directory](go), run `go build -o ../bin/nba_main .`

### **Python** 

Python files are under the [python directory](python). Ensure relevant packages are installed with `pip install -r requirements.txt`

### **Airflow**

We use airflow to handle the data sourcing process. The process runs once, nightly, and collects data for the T+2 date. This lets us collect data throughout a season with minimal manual activity.
1. Install airflow if not already installed - https://airflow.apache.org/docs/apache-airflow/stable/start.html
2. Set `AIRFLOW_HOME` env variable as the absolute path to [/airflow](airflow)
3. `airflow users create --username admin --firstname test --lastname admin --role Admin --email admin@email.com` - can use any other username, password combo
4. `airflow db init` 
5. in one terminal: `airflow scheduler` 
6. in another: `airflow webserver -p 8080`
7. Update [nba_project_dag.py](airflow/dags/nba_project_dag.py), setting variables `PYTHON_PATH` and `PROJECT_HOME`

That's it. Heading to http://localhost:8080/home should bring up the airflow UI, where we can trigger **nba_project_dag**. 

### **Running tasks individually** 

If the airflow setup worked, this section can be skipped. If there are issues with airflow, or if we need to run the tasks manually, we can trigger each sourcing job individually. For the golang jobs, we specify the process through a command line argument. This is order they should be run: 
1. fetch games (python): `python python/raw_game_data_sourcing.py 2024-10-24`
2. fetch odds (go): `bin/nba_main --config=go/go_config.yaml --date=2024-10-24 --process=fetch_raw_odds`
3. clean games (go): `bin/nba_main --config=go/go_config.yaml --date=2024-10-24 --process=clean_games`
4. clean odds (go): `bin/nba_main --config=go/go_config.yaml --date=2024-10-24 --process=clean_raw_odds`
5. combine games and odds to csv (go): `bin/nba_main --config=go/go_config.yaml --date=2024-10-24 --process=combine_game_and_odds`

### **Analyzing data** 

Once we've done our data sourcing and poulated the csvs, we can run the script [historical_analysis.py](python/historical_analysis.py) to give us answers - in the form of historical results - to the questions above. To set a specific scenario, i.e. team X has a 15 point lead in with 6:00 to go in the third, we can set the filters defined in [analysis_config.py](python/analysis_config.py.py). These filters include both pregame and ingame margins, and are also team and date specific. This approach is similar to the one defined in [this blog post](https://plusevanalytics.wordpress.com/2024/02/02/sampling-using-tightness-and-boost/), but with the ability to use in game scenarios.
