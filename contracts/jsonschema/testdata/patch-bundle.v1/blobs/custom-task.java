package com.example.build;

import org.gradle.api.DefaultTask;
import org.gradle.api.tasks.InputFiles;
import org.gradle.api.tasks.OutputDirectory;

public abstract class BundleFrontend extends DefaultTask {
    @InputFiles
    public abstract Object getSourceFiles();

    @OutputDirectory
    public abstract Object getOutputDirectory();
}
