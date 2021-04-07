package storage_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
)

type StorageDataLakeGen2PathDataSoure struct{}

func TestAccDataSourceStorageAccountDataLakeGen2Path_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "data.azurerm_storage_data_lake_gen2_path", "test")

	data.DataSourceTest(t, []resource.TestStep{
		{
			Config: StorageAccountDataSource{}.basicWithDataSource(data),
		},
	})
}

func (d StorageDataLakeGen2PathDataSoure) basic(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-storage-%d"
  location = "%s"
}

resource "azurerm_storage_account" "test" {
  name                = "acctestsads%s"
  resource_group_name = azurerm_resource_group.test.name

  location                 = azurerm_resource_group.test.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
  account_kind			   = "StorageV2"
  is_hns_enabled           = "true"

  tags = {
    environment = "production"
  }
}


  
  resource "azurerm_storage_data_lake_gen2_filesystem" "example" {
	name               = "example"
	storage_account_id = azurerm_storage_account.test.id
  }

  resource "azurerm_storage_data_lake_gen2_path" "root" {
	path               = "root"
	filesystem_name    = azurerm_storage_data_lake_gen2_filesystem.example.name
	storage_account_id = azurerm_storage_account.example.id
	resource           = "directory"

	ace {
		scope = "default"
		type = "user"
		id = "0e146308-cd0c-4a59-a5b5-d86be09aecee"
		permission = "r--"
	}
  }

  resource "azurerm_storage_data_lake_gen2_path" "child" {
	path               = "root/child"
	filesystem_name    = azurerm_storage_data_lake_gen2_filesystem.example.name
	storage_account_id = azurerm_storage_account.example.id
	resource           = "directory"

	ace {
		scope = "access"
		type = "user"
		id = "e83a0d94-d465-467c-9a1d-cef2adc45365"
		permission = "rwx"
	}
  }
`, data.RandomInteger, data.Locations.Primary, data.RandomString)
}

func (d StorageDataLakeGen2PathDataSoure) basicWithDataSource(data acceptance.TestData) string {
	config := d.basic(data)
	return fmt.Sprintf(`
%s

data "azurerm_storage_data_lake_gen2_path" "test" {
  name                = azurerm_storage_data_lake_gen2_path.child.id
}
`, config)
}
