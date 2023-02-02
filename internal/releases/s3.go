package releases

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func S3Schema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"access_key_id": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "S3 Access Key ID",
			},
			"secret_access_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "S3 Secret Access Key",
			},
			"paths": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				Elem:        PackagePathsSchema(),
				Description: "Package path and checksum",
			},
		},
	}
}

func formatInputS3(ctx context.Context, data []interface{}) map[string]interface{} {

	s3 := make(map[string]interface{})
	for _, v := range data {
		s3 = v.(map[string]interface{})
		s3["accessKeyId"] = s3["access_key_id"]
		delete(s3, "access_key_id")
		s3["secretAccessKey"] = s3["secret_access_key"]
		delete(s3, "secret_access_key")
		s3["paths"] = formatInputPaths(ctx, s3["paths"])

	}
	return s3
}

func formatOutputS3(ctx context.Context, s3 map[string]interface{}) []map[string]interface{} {

	s3["access_key_id"] = s3["accessKeyId"]
	delete(s3, "accessKeyId")
	s3["secret_access_key"] = s3["secretAccessKey"]
	delete(s3, "secretAccessKey")
	mapSlice := []map[string]interface{}{}
	s3["paths"] = append(mapSlice, s3["paths"].(map[string]interface{}))

	s3_formatted := []map[string]interface{}{}
	s3_formatted = append(s3_formatted, s3)
	return s3_formatted
}