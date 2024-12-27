from airflow import DAG
from airflow.operators.bash import BashOperator
from datetime import datetime, timedelta

# TODO: Populate these 
# Path to python installation .../bin/python3
PYTHON_PATH = ''
# Absolute path to the root directory of the project .../nba-live-scoring-analysis
PROJECT_HOME = ''

PYTHON_SCRIPTS_HOME = f'{PROJECT_HOME}/python'
CONFIG_FILE_PATH = 'go/go_config.yaml'

GO_PARAMS = {
    'home': PROJECT_HOME, 
    'config': CONFIG_FILE_PATH 
}

default_args = {
    'owner': 'airflow',
    'retries': 3,
    'retry_delay': timedelta(minutes=5),    
    'start_date': datetime(2024, 10, 22),
}

with DAG(
    dag_id='nba_project_dag',
    default_args=default_args,
    schedule_interval=timedelta(days=1),
    catchup=False,
) as dag:
    
    fetch_games_task = BashOperator(
        task_id='fetch_games_task',
        bash_command='cd {{ params.python_dir}} && {{ params.python_path }} raw_game_data_sourcing.py {{ ds }}',
        params={ 
            'python_dir': PYTHON_SCRIPTS_HOME,
            'python_path': PYTHON_PATH 
        }
    )
    
    fetch_odds_task = BashOperator(
        task_id='fetch_raw_odds',
        bash_command='cd {{ params.home }} && bin/nba_main --process=fetch_raw_odds --date={{ ds }} --config={{ params.config }}',
        env={ 'PATH': '/usr/local/go/bin'},
        params=GO_PARAMS
    )

    clean_games_task = BashOperator(
        task_id='clean_games',
        bash_command='cd {{ params.home }} && bin/nba_main --process=clean_games --date={{ ds }} --config={{ params.config }}',
        env={ 'PATH': '/usr/local/go/bin'},
        params=GO_PARAMS
    )

    clean_odds_task = BashOperator(
        task_id='clean_odds_task',
        bash_command='cd {{ params.home }} && bin/nba_main --process=clean_raw_odds --date={{ ds }} --config={{ params.config }}',
        env={ 'PATH': '/usr/local/go/bin'},
        params=GO_PARAMS
    )
    
    combine_games_and_odds_task = BashOperator(
        task_id='combine_games_and_odds_task',
        bash_command='cd {{ params.home }} && bin/nba_main --process=combine_game_and_odds --date={{ ds }} --config={{ params.config }}',
        env={ 'PATH': '/usr/local/go/bin'},
        params=GO_PARAMS
    )
    
    fetch_games_task.set_downstream(clean_games_task)
    fetch_odds_task.set_downstream(clean_odds_task)
    clean_games_task.set_downstream(clean_odds_task)
    clean_odds_task.set_downstream(combine_games_and_odds_task)
    