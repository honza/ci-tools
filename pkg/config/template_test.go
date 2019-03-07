package config

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	templateapi "github.com/openshift/api/template/v1"
	templatescheme "github.com/openshift/client-go/template/clientset/versioned/scheme"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/fake"
	coretesting "k8s.io/client-go/testing"
)

const templatesPath = "../../test/pj-rehearse-integration/master/ci-operator/templates"

func TestGetTemplates(t *testing.T) {
	expectCiTemplates := getBaseCiTemplates(t)
	if templates, err := getTemplates(templatesPath); err != nil {
		t.Fatalf("getTemplates() returned error: %v", err)
	} else if !equality.Semantic.DeepEqual(templates, expectCiTemplates) {
		t.Fatalf("Diff found %s", diff.ObjectReflectDiff(expectCiTemplates, templates))
	}
}

func TestCreateCleanupCMTemplates(t *testing.T) {
	expectedCmName := "rehearse-1234-test-template"
	expectedCmLabels := map[string]string{
		createByRehearse:  "true",
		rehearseLabelPull: "1234",
	}

	createByRehearseReq, err := labels.NewRequirement(createByRehearse, selection.Equals, []string{"true"})
	if err != nil {
		t.Fatal(err)
	}

	rehearseLabelPullReq, err := labels.NewRequirement(rehearseLabelPull, selection.Equals, []string{"1234"})
	if err != nil {
		t.Fatal(err)
	}

	selector := labels.NewSelector().Add(*createByRehearseReq).Add(*rehearseLabelPullReq)

	expectedListRestricitons := coretesting.ListRestrictions{
		Labels: selector,
	}

	cs := fake.NewSimpleClientset()
	cs.Fake.AddReactor("create", "configmaps", func(action coretesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(coretesting.CreateAction)
		cm := createAction.GetObject().(*v1.ConfigMap)

		if cm.ObjectMeta.Name != expectedCmName {
			t.Fatalf("Configmap name:\nExpected: %s\nFound: %s", expectedCmName, cm.ObjectMeta.Name)
		}

		if !reflect.DeepEqual(cm.ObjectMeta.Labels, expectedCmLabels) {
			t.Fatalf("Configmap labels\nExpected: %#v\nFound: %#v", expectedCmLabels, cm.ObjectMeta.Labels)
		}

		return true, nil, nil
	})
	cs.Fake.AddReactor("delete-collection", "configmaps", func(action coretesting.Action) (handled bool, ret runtime.Object, err error) {
		deleteAction := action.(coretesting.DeleteCollectionAction)
		listRestricitons := deleteAction.GetListRestrictions()

		if !reflect.DeepEqual(listRestricitons.Labels, expectedListRestricitons.Labels) {
			t.Fatalf("Labels:\nExpected:%#v\nFound: %#v", expectedListRestricitons.Labels, listRestricitons.Labels)
		}

		return true, nil, nil
	})

	ciTemplates := getBaseCiTemplates(t)
	cmManager := NewTemplateCMManager(cs.CoreV1().ConfigMaps("test-namespace"), 1234, logrus.NewEntry(logrus.New()), ciTemplates)

	if err := cmManager.CreateCMTemplates(); err != nil {
		t.Fatalf("CreateCMTemplates() returned error: %v", err)
	}

	if err := cmManager.CleanupCMTemplates(); err != nil {
		t.Fatalf("CleanupCMTemplates() returned error: %v", err)
	}

}

func getBaseCiTemplates(t *testing.T) CiTemplates {
	testTemplatePath := filepath.Join(templatesPath, "test-template.yaml")
	contents, err := ioutil.ReadFile(testTemplatePath)
	if err != nil {
		t.Fatalf("could not read file %s for template: %v", testTemplatePath, err)
	}

	var expectedTemplate *templateapi.Template
	if obj, _, err := templatescheme.Codecs.UniversalDeserializer().Decode(contents, nil, nil); err == nil {
		if template, ok := obj.(*templateapi.Template); ok {
			if len(template.Name) == 0 {
				template.Name = filepath.Base(testTemplatePath)
				template.Name = strings.TrimSuffix(template.Name, filepath.Ext(template.Name))
			}
			expectedTemplate = template
		}
	}

	return CiTemplates{
		"test-template.yaml": expectedTemplate,
	}
}
