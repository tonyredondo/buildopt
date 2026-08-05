plugins {
    java
}

val discoverySelfReference by configurations.creating

dependencies {
    implementation(project(":library-c"))
    add(discoverySelfReference.name, project(":service-a"))
}
