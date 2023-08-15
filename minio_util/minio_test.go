package minio_util

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"

	"github.com/DoOR-Team/goutils/log"
)

func LoadViperFromFiles(files ...string) error {
	viper.SetConfigType("yaml")
	for _, file := range files {
		// 读取配置文件
		if file == "" {
			log.Panic("配置文件不能为空")
		}
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		err = viper.MergeConfig(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestMinIo(t *testing.T) {
	_ = LoadViperFromFiles("config.yaml", "secret.yaml")

	ctx := context.Background()
	endpoint := viper.GetString("minio_path")
	accessKeyID := viper.GetString("minio_access_key")
	secretAccessKey := viper.GetString("minio_access_secret")
	useSSL := true

	// Initialize minio client object.
	log.Info(endpoint)
	// log.Info(accessKeyID)

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Make a new bucket called mymusic.
	bucketName := viper.GetString("k8s_namespace") + "." + viper.GetString("bucket_name")

	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatal(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucketName)
	}

	// Upload the zip file
	objectName := "gou.jpeg"
	filePath := "/Users/jpbirdy/Downloads/gou.jpeg"
	// contentType := "application/zip"

	// Upload the zip file with FPutObject
	n, err := minioClient.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Successfully uploaded %s of size %v", objectName, n)

	url, err := minioClient.PresignedGetObject(ctx, bucketName, objectName, time.Second*10*60, make(url.Values))

	log.Println(url)
}

func TestMinio2(t *testing.T) {
	_ = LoadViperFromFiles("config.yaml", "secret.yaml")

	ctx := context.Background()
	endpoint := viper.GetString("minio_path")
	accessKeyID := viper.GetString("minio_access_key")
	secretAccessKey := viper.GetString("minio_access_secret")
	useSSL := true

	// Initialize minio client object.
	log.Info(endpoint)
	// log.Info(accessKeyID)

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatal(err)
	}

	bucketName := "preform-data-center"
	objectName := "2020/12/21/20201221_d-01_r082518_a48p0232-45b/data.xls"

	url, err := minioClient.PresignedGetObject(ctx, bucketName, objectName, time.Second*10*60, make(url.Values))

	log.Println(url, err)
}
