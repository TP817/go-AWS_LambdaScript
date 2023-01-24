package main

import (
    "fmt"
    "context"
    "bytes"
    "strings"

    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go/service/sts"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func UploadFile(s *session.Session, bucketName string, objectKey string, payload [][]string) error {
    body := ""
    for _, row := range payload {
        for i:=0; i<len(row); i ++ {
            body = body + row[i];
            if i >= (len(row)-1) {
                body = body + "\n"
            } else {
                body = body + ","
            }
        }
    }

    _, err := s3.New(s).PutObject(&s3.PutObjectInput{
        Bucket: aws.String(bucketName),
        Key:    aws.String(objectKey),
        Body:   bytes.NewReader([]byte(body)),
    })

    return err
}

func ListBuckets(client *s3.S3) (*s3.ListBucketsOutput, error) {
    res, err := client.ListBuckets(nil)
    if err != nil {
        return nil, err
    }

    return res, nil
}

func HandleRequest() {
    sess := session.Must(session.NewSessionWithOptions(session.Options{
        SharedConfigState: session.SharedConfigEnable,
    }))

    // conf := aws.NewConfig().WithRegion("us-east-2")

    s3Client := s3.New(sess)
    // bucketManager := s3manager.New(sess, conf)
    // title := currentTime.Format("2006-01-02")

    var payload [][]string;
    payload = append(payload, []string{ "BucketName", "Policy", "AccountID", "Region"})

    buckets, err := ListBuckets(s3Client)
    if err != nil {
        fmt.Println("Error", err)
    }

    for _, bucket := range buckets.Buckets {
        region, err := s3manager.GetBucketRegion(context.TODO(), sess, *bucket.Name, "us-east-2")
        if err != nil {
            fmt.Println("Error", err)
        }
 
        bucketName := *bucket.Name

        conf := aws.NewConfig().WithRegion(region)

        s3Temp := s3.New(sess, conf)

        input := &s3.GetBucketLifecycleConfigurationInput{
            Bucket: aws.String(bucketName),
        }
        result, err := s3Temp.GetBucketLifecycleConfiguration(input)
        rules := []*s3.LifecycleRule{}
        if err == nil {
            rules = result.Rules   
        }

        regionConf := aws.NewConfig().WithRegion(region)
        stsConn := sts.New(sess, regionConf)
        outCallerIdentity, err := stsConn.GetCallerIdentity(&sts.GetCallerIdentityInput{})
        if err != nil {
            fmt.Println("Error", err)
        }
        accountID := *outCallerIdentity.Account

        if len(rules) == 0 {
            payload = append(payload, []string{
                        *bucket.Name,
                        "No Rule",
                        "" + accountID + "",
                        region,
                    })
        } else {
            for _, rule := range rules {
                stringRule := rule.GoString()
                payload = append(payload, []string{
                            *bucket.Name,
                            stringRule,
                            "" + accountID + "",
                            region,
                        })
            }
        } 
    }
    err = UploadFile(sess, "monitoring-v0", "S3Bucket.csv",  payload)
    if err != nil {
        fmt.Println("Error", err)
    }
}

func main() {
    lambda.Start(HandleRequest)
}
