package storage

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/storage/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
	"github.com/tombuildsstuff/giovanni/storage/2019-12-12/datalakestore/paths"
	"github.com/tombuildsstuff/giovanni/storage/accesscontrol"
)

func dataSourceStorageDataLakeGen2Path() *schema.Resource {
	return &schema.Resource{
		Read: dataStorageDataLakeGen2PathRead,

		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"ace_scope_filter": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "all",
				ValidateFunc: validation.StringInSlice([]string{"all", "default", "access"}, false),
			},
			"storage_account_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validate.StorageAccountID,
			},

			"filesystem_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateStorageDataLakeGen2FileSystemName,
			},

			"path": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"resource": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"directory"}, false),
			},

			"owner": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IsUUID,
			},

			"group": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IsUUID,
			},

			"ace": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"scope": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.StringInSlice([]string{"default", "access"}, false),
						},
						"type": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.StringInSlice([]string{"user", "group", "mask", "other"}, false),
						},
						"id": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.IsUUID,
						},
						"permissions": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validate.ADLSAccessControlPermissions,
						},
					},
				},
			},
		},
	}
}

func dataStorageDataLakeGen2PathRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Storage.ADLSGen2PathsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := paths.ParseResourceID(d.Get("id").(string))
	if err != nil {
		return err
	}

	aceFilter := d.Get("ace_scope_filter").(string)

	log.Printf("[INFO] Id Path is: %s", id.Path)
	log.Printf("[INFO] Id Filesystem is: %s", id.FileSystemName)
	log.Printf("[INFO] Id Account Name is: %s", id.AccountName)

	resp, err := client.GetProperties(ctx, id.AccountName, id.FileSystemName, id.Path, paths.GetPropertiesActionGetStatus)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] Path %q does not exist in File System %q in Storage Account %q - removing from state...", id.Path, id.FileSystemName, id.AccountName)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Path %q in File System %q in Storage Account %q: %+v", id.Path, id.FileSystemName, id.AccountName, err)
	}

	d.SetId(d.Get("id").(string))
	d.Set("path", id.Path)
	d.Set("filesystem_name", id.FileSystemName)
	d.Set("storage_account_id", id.AccountName)
	d.Set("resource", resp.ResourceType)
	d.Set("owner", resp.Owner)
	d.Set("group", resp.Group)

	log.Printf("[INFO] Owner is: %s", resp.Owner)
	log.Printf("[INFO] Group is: %s", resp.Group)

	// The above `getStatus` API request doesn't return the ACLs
	// Have to make a `getAccessControl` request, but that doesn't return all fields either!
	resp, err = client.GetProperties(ctx, id.AccountName, id.FileSystemName, id.Path, paths.GetPropertiesActionGetAccessControl)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] Path %q does not exist in File System %q in Storage Account %q - removing from state...", id.Path, id.FileSystemName, id.AccountName)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving ACLs for Path %q in File System %q in Storage Account %q: %+v", id.Path, id.FileSystemName, id.AccountName, err)
	}

	acl, err := accesscontrol.ParseACL(resp.ACL)
	if err != nil {
		return fmt.Errorf("Error parsing response ACL %q: %s", resp.ACL, err)
	}

	flattenedACLs := FlattenDataLakeGen2AceList(acl)

	// If the scope is set to anything other than all, filter out all the unwanted scopes
	if aceFilter != "all" {
		log.Printf("[INFO] Filtering ACL's to: %s", aceFilter)
		flattenedACLs = filterAceByScope(flattenedACLs, aceFilter)
	}

	d.Set("ace", flattenedACLs)

	return nil
}

func filterAceByScope(ace []interface{}, scope string) []interface{} {
	filteredAces := []interface{}{}

	for _, value := range ace {
		current := value.(map[string]interface{})

		if current["scope"] == scope {
			filteredAces = append(filteredAces, current)
		}
	}

	return filteredAces

}
