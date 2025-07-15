#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { KinesisStack } from './stacks/kinesis-stack';
import * as dotenv from 'dotenv';

// Load environment variables from .env file
dotenv.config();

const app = new cdk.App();

// Environment configuration
const env = {
  account: process.env.CDK_DEFAULT_ACCOUNT || process.env.AWS_ACCOUNT_ID,
  region: process.env.CDK_DEFAULT_REGION || process.env.AWS_REGION || 'us-east-1',
};

// Tags for all resources
const tags = {
  Project: 'Payloop',
  Environment: process.env.ENVIRONMENT || 'dev',
  ManagedBy: 'CDK',
};

// Create the Kinesis stack
new KinesisStack(app, 'PayloopKinesisStack', {
  env,
  description: 'Payloop Kinesis streams for event processing',
  tags,
});

app.synth();