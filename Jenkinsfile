pipeline {
  agent any

  stages {
    stage('Checkout') {
      steps {
        step([$class: 'WsCleanup'])
        checkout(scm)
      }
    }
    stage('Build Image') {
      steps {
        sh('make build-image')
      }
    }
    stage('Tag and Push Image') {
      steps {
        sh('make push')
      }
    }
  }
}