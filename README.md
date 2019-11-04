### Argo Scheduler

Leverages Argo Workflows to schedule or watch for events that trigger Workflows to launch.

Workflows are stored in GitHub. 

0. Argo Scheduler use Kubernetes Curd and kubectl.
1. Workflows stored in GitHub. 
        Folows GitOps model. Workflow changes can use Pull-Request model. 
        Support for GitHub Enterprise with tokens.
2. Access Uer and Access Password will be entered using Environment variables initialy with
   future support for Kubernetes Secrets.
3. Scheduled Workflows. 
        Uses familiar cron sintax to define when Workflows should be executed.
4. File triggered Workflows
        A watched over S3, FTP. Once a file is detected (and optionaly if it has a age) Will trigger
        a workflow. 
        - issue: Control frecuency. by Marking in memory workflow is triggered and upon completition
          update, rename or delete trigger file (this needs write access to the watch file system.
          Additionaly we can provide a custom http: trigger upon completition. This could be a rest service
          that will be in charge of removing triggers.
5. Workflows are run natively using go code. to interact with Kubernetes.

Extra:

1. Store workflow execution history in MySQL for future reporting.
2. Record Workflow execution output in MySQL Blob for future usage.
3. Record Workflow run times to create detailed run history reports.
4. Some simple UI for demo.
5. EFS file watcher.
6. use go text/template in workflows before submit to make them more flexible.  Available variables will be related to the scheduler
        - Job name
        - Execution run of the day
        - Original scheduled time
        - today
        - Today-1
        - workday - 1  (if today is Monday it will return Friday)
