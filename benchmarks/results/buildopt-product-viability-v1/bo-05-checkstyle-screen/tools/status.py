"""Read only the sixteen owned attempt receipts and recent output."""
from pathlib import Path
import datetime
import json

P = Path(__file__).resolve().parent
run = P / 'profiles/supported/run'
print(datetime.datetime.now(datetime.timezone.utc).isoformat())
starts = list((run/'costs').glob('*.start.json'))
pending = [p.name for p in starts if not p.with_name(p.name.replace('.start.json','.json')).exists()
           and not p.name.endswith('.work-start.json')]
print('Active phases:', pending)
for attempt in sorted((run/'attempts').glob('*')):
    files = [name for name in ('native-start.json','native-finish.json','end.json') if (attempt/name).exists()]
    print(attempt.name, files)
    if (attempt/'native-finish.json').exists():
        finish = json.loads((attempt/'native-finish.json').read_text())
        print('native exit',finish['exitCode'],'seconds',(finish['end']['ns']-finish['start']['ns'])/1e9)
    elif (attempt/'stdout.log').exists():
        log = attempt/'stdout.log'
        with log.open('rb') as output:
            output.seek(max(0,log.stat().st_size-400))
            print(output.read().decode(errors='replace')[-300:])
print('Completed pairs:',len(list((run/'pairs').glob('*.json'))),'Result:',(run/'result.json').exists())
