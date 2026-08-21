plugins {
    java
}

val discoverySelfReference by configurations.creating
val inheritedProjectDependencies by configurations.creating

configurations.implementation {
    extendsFrom(inheritedProjectDependencies)
}

dependencies {
    add(inheritedProjectDependencies.name, project(":library-c"))
    add(discoverySelfReference.name, project(":service-a"))
}
