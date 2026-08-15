# Rejected pre-measurement attempt: Ktor target-property semantics

The fourth fresh attempt passed BuildOpt's generic CLI validation and began the
untimed output-contract workflow, but Ktor scheduled native `assemble` work even
though the command included `-Ptarget.*=false` options. Inspection of Ktor's
public build logic showed that its `ProjectGradleProperties` value source reads
`gradle.properties` files directly and does not consult Gradle CLI project
properties. The preregistered JVM-only description was therefore false.

The process was stopped before proposal completion, warm-up or timing. No
output, duration or partial task result is accepted. A non-measured
`./gradlew help --task jvmJar` compatibility check then confirmed Ktor's public
JVM JAR selector across its subprojects. The preregistration is amended to that
selector without changing the repository revision, fixed source mutation,
required JAR, qualification gates or generic Build Impact mechanism.
