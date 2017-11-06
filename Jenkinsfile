#!groovy
node("Ubuntu1604") {
    deleteDir()
    currentBuild.result = "SUCCESS"

    ECR_CREDENTIALS_ID_1 = "Jenkins-cbo-ecr-user"
    ECR_CREDENTIALS_ID_2 = "cbo-jenkins-ecs-push-prod"
    ECR_REGION_1 = "eu-central-1"
    ECR_REGION_2 = "eu-west-1"
    ECR_USER_1 = "814269883889"
    ECR_USER_2 = "304610113237"
    ECR_DOMAIN_1 = "${ECR_USER_1}.dkr.ecr.${ECR_REGION_1}.amazonaws.com"
    ECR_DOMAIN_2 = "${ECR_USER_2}.dkr.ecr.${ECR_REGION_2}.amazonaws.com"
    CONTAINER_IMAGE_NAME = "imageproxy"

    JDK_VERSION = "Java SDK 8"

    try {
        stage("Checkout GIT") {
            checkout scm
        }

        stage("Maven Clean Install") {
            withCredentials([[$class: 'AmazonWebServicesCredentialsBinding', accessKeyVariable: 'AWS_ACCESS_KEY_ID', credentialsId: 'cbo-api-documentation', secretKeyVariable: 'AWS_SECRET_ACCESS_KEY']]) {
                withMaven(globalMavenSettingsConfig: GLOBAL_MAVEN_SETTINGS_ID, jdk: JDK_VERSION, maven: MAVEN, mavenSettingsConfig: MAVEN_SETTINGS_ID) {
                    sh "export AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID; export AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY; mvn clean install -U"
                }
            }
        }

        stage("ECR build/push ${ECR_REGION_1}") {
            if (params.VERSION == 'LATEST') {
                gitCommit = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
                image_name = "${ECR_DOMAIN_1}/${CONTAINER_IMAGE_NAME}:${gitCommit}-${env.BUILD_NUMBER}"
            } else {
                image_name = "${ECR_DOMAIN_1}/${CONTAINER_IMAGE_NAME}:${params.VERSION}"
            }
            docker.build(image_name)
            docker.withRegistry("https://${ECR_DOMAIN_1}", "ecr:${ECR_REGION_1}:${ECR_CREDENTIALS_ID_1}") {
                docker.image(image_name).push()
            }
        }

        stage("ECR build/push ${ECR_REGION_2}") {
            if (params.VERSION == 'LATEST') {
                gitCommit = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
                image_name = "${ECR_DOMAIN_2}/${CONTAINER_IMAGE_NAME}:${gitCommit}-${env.BUILD_NUMBER}"
            } else {
                image_name = "${ECR_DOMAIN_2}/${CONTAINER_IMAGE_NAME}:${params.VERSION}"
            }

            docker.build(image_name)
            docker.withRegistry("https://${ECR_DOMAIN_2}", "ecr:${ECR_REGION_2}:${ECR_CREDENTIALS_ID_2}") {
                docker.image(image_name).push()
            }
        }

        stage("Slack Notification") {
            if (currentBuild.result != currentBuild.getPreviousBuild()?.getResult()) {
                withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: '681b12dd-880e-4c69-ba72-dfb6148eece3', passwordVariable: 'SLACK_TOKEN', usernameVariable: 'teamDomain']]) {
                    slackSend channel: '#green', message: 'The ' + JOB_NAME + ' build changed state to ' + currentBuild.result + '...!', teamDomain: 'merapar', token: SLACK_TOKEN
                }
            }
        }
    } catch (err) {
        currentBuild.result = "FAILURE"
        withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: '681b12dd-880e-4c69-ba72-dfb6148eece3', passwordVariable: 'SLACK_TOKEN', usernameVariable: 'teamDomain']]) {
            slackSend channel: '#green', message: 'The ' + JOB_NAME + ' build encountered an error...!', teamDomain: 'merapar', token: SLACK_TOKEN
        }
        throw err
    }
}
