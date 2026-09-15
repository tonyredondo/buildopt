"""Summarize recorded disk supervision without new native commands."""
from pathlib import Path
import json

root = Path(__file__).resolve().parents[1]
def read(path):
    return json.loads(path.read_text())

rows = []
stages = []
for profile in sorted((root / "profiles").iterdir()):
    run = profile / "run"
    if not run.exists():
        continue
    for path in sorted((run / "attempts").glob("*/native-finish.json")):
        native = read(path)
        cpu = read(path.parent / "supervisor-cpu.json")
        disk = read(path.parent / "disk-observer.json")
        seconds = (native["end"]["ns"] - native["start"]["ns"]) / 1e9
        cpu_seconds = sum(cpu["end"][key] - cpu["begin"][key] for key in ("userNS", "systemNS")) / 1e9
        assert seconds > 0 and cpu_seconds >= 0
        assert disk["attempt"] == native["attempt"]
        rows.append(dict(profile=profile.name, attempt=path.parent.name,
                         nativeSeconds=seconds, supervisorCPUSeconds=cpu_seconds,
                         supervisorCoreEquivalent=cpu_seconds / seconds,
                         disk=disk["status"]))
    samples = 0
    collection_cpu = 0
    collection_elapsed = 0
    unavailable_host = 0
    unavailable_group = 0
    group_observations = 0
    for line in (profile / "process-samples.jsonl").open():
        sample = json.loads(line)["storage"]
        samples += 1
        collection_cpu += sample["samplerAfter"]["selfCPUNS"] - sample["samplerBefore"]["selfCPUNS"]
        collection_elapsed += sample["endBootNS"] - sample["beginBootNS"]
        unavailable_host += any(not sample["host"][key]["available"] for key in ("deviceCounters", "pendingWrites"))
        for group in sample["groups"].values():
            group_observations += 1
            unavailable_group += any(not group[key]["available"] for key in ("ioPressure", "ioCounters"))
    end = read(root / "receipts" / (profile.name + "-end.json"))
    stages.append(dict(profile=profile.name, samples=samples,
                       storageCollectionCPUSeconds=collection_cpu / 1e9,
                       storageCollectionElapsedSeconds=collection_elapsed / 1e9,
                       wholeSamplerCPUSeconds=(end["samplerAfter"]["selfCPUNS"]-end["samplerBefore"]["selfCPUNS"]) / 1e9,
                       unavailableHostSamples=unavailable_host,
                       groupObservations=group_observations,
                       unavailableGroupObservations=unavailable_group))
result = dict(requests=rows, sampler=stages,
              interpretation="Supervisor CPU includes initial observer setup and all other supervision. "
              "Setup is not separately timed. Host counters do not identify an external cause. "
              "These observations do not establish the effect of supervision on build wall time.")
with (root / "analysis/disk-observations.json").open("x") as output:
    json.dump(result, output, indent=2)
    output.write("\n")
print(json.dumps(result))
