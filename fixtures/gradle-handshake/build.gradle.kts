import org.gradle.api.tasks.WriteProperties

plugins {
    base
}

tasks.register<WriteProperties>("neutralProbe") {
    destinationFile = layout.buildDirectory.file("ws-003/neutral.properties")
    property("marker", "ws-003-neutral")
}

tasks.register<Exec>("intentionalFailure") {
    commandLine(
        "sh",
        "-c",
        "printf '%s\\n' 'WS-003 intentional baseline failure' >&2; exit 1",
    )
}
