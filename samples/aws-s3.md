# Using AWS S3 from a function
This is not a fully working application, but just illustrates how to use the AWS SDK from a function.
The runner docker image comes with `aws-sdk` pre-loaded, that's why you can simply import it.

## Environment
The names of these variables are important, they're picked up by the AWS SDk.
```yaml
AWS_ACCESS_KEY_ID: YOUR_KEY_HERE
AWS_SECRET_ACCESS_KEY: YOUR_ACCESS_KEY
```

## Function: TestS3
This does nothing useful, just illustrates how to set things up and make a call.
```javascript
import AWS from 'aws-sdk';

AWS.config.update({region: 'eu-central-1'});

async function handle() {
    var s3 = new AWS.S3();
    let result = await s3.listObjects({Bucket: "my-bucket"}).promise()
    console.log(result);
}
```